package postgres

import (
	"context"
	"doodhee-backend/internal/domain"
	"doodhee-backend/internal/ports"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"sort"
	"strings"
)

// Pool and transactions are selected explicitly to keep transaction boundaries visible.
type Store struct {
	Pool *pgxpool.Pool
	tx   pgx.Tx
}

var tables = map[string]bool{"users": true, "addresses": true, "products": true, "orders": true, "milkman_customers": true, "customer_product_prices": true, "extra_milk_requests": true, "milkman_ratings": true, "auth_challenges": true, "auth_sessions": true, "mfa_methods": true, "outbox_events": true, "admin_audit_logs": true, "idempotency_keys": true, "notifications": true}

func (s *Store) query(ctx context.Context, q string, args ...any) (pgx.Rows, error) {
	if s.tx != nil {
		return s.tx.Query(ctx, q, args...)
	}
	return s.Pool.Query(ctx, q, args...)
}
func (s *Store) Ping(ctx context.Context) error { return s.Pool.Ping(ctx) }
func (s *Store) Lock(ctx context.Context, key string) error {
	if s.tx == nil {
		return fmt.Errorf("lock requires transaction")
	}
	_, e := s.tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", key)
	return e
}
func (s *Store) Tx(ctx context.Context, fn func(ports.Store) error) error {
	if s.tx != nil {
		return fn(s)
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if e = fn(&Store{Pool: s.Pool, tx: tx}); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func rowsJSON(rows pgx.Rows) ([]domain.Record, error) {
	defer rows.Close()
	out := []domain.Record{}
	for rows.Next() {
		var raw []byte
		if e := rows.Scan(&raw); e != nil {
			return nil, e
		}
		var r domain.Record
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.UseNumber()
		if e := dec.Decode(&r); e != nil {
			return nil, e
		}
		for _, k := range []string{"price", "daily_capacity", "quantity", "base_quantity", "approved_extra_quantity", "extra_quantity", "unit_price", "total_amount", "average_rating"} {
			if r[k] != nil {
				r[k] = fmt.Sprint(r[k])
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func (s *Store) Get(ctx context.Context, table, id string, lock bool) (domain.Record, error) {
	if !tables[table] {
		return nil, fmt.Errorf("invalid table")
	}
	q := "SELECT to_jsonb(t) FROM " + table + " t WHERE id::text=$1"
	if lock {
		q += " FOR UPDATE"
	}
	rows, e := s.query(ctx, q, id)
	if e != nil {
		return nil, e
	}
	list, e := rowsJSON(rows)
	if e != nil {
		return nil, e
	}
	if len(list) == 0 {
		return nil, domain.Fail(404, "NOT_FOUND", "The record was not found or is not accessible.")
	}
	return list[0], nil
}
func (s *Store) List(ctx context.Context, table string, filters domain.Record) ([]domain.Record, error) {
	if !tables[table] {
		return nil, fmt.Errorf("invalid table")
	}
	args := []any{}
	where := []string{}
	keys := []string{}
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if strings.ContainsAny(k, "; -\"'") {
			return nil, fmt.Errorf("invalid filter")
		}
		args = append(args, filters[k])
		where = append(where, fmt.Sprintf("%s::text=$%d", pgx.Identifier{k}.Sanitize(), len(args)))
	}
	q := "SELECT to_jsonb(t) FROM " + table + " t"
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY id"
	rows, e := s.query(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	return rowsJSON(rows)
}
func (s *Store) Save(ctx context.Context, table string, r domain.Record) (domain.Record, error) {
	if !tables[table] {
		return nil, fmt.Errorf("invalid table")
	}
	keys := []string{}
	for k := range r {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	cols, vals, updates := []string{}, []string{}, []string{}
	args := []any{}
	for i, k := range keys {
		col := pgx.Identifier{k}.Sanitize()
		cols = append(cols, col)
		vals = append(vals, fmt.Sprintf("$%d", i+1))
		args = append(args, r[k])
		if k != "id" && k != "created_at" {
			updates = append(updates, col+"=EXCLUDED."+col)
		}
	}
	q := "INSERT INTO " + table + " (" + strings.Join(cols, ",") + ") VALUES (" + strings.Join(vals, ",") + ") ON CONFLICT(id) DO UPDATE SET " + strings.Join(updates, ",") + " RETURNING to_jsonb(" + table + ".*)"
	rows, e := s.query(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	list, e := rowsJSON(rows)
	if e != nil {
		return nil, e
	}
	return list[0], nil
}

func (s *Store) Delete(ctx context.Context, table, id string) error {
	if !tables[table] {
		return fmt.Errorf("invalid table")
	}
	q := "DELETE FROM " + table + " WHERE id::text=$1"
	var result pgconn.CommandTag
	var e error
	if s.tx != nil {
		result, e = s.tx.Exec(ctx, q, id)
	} else {
		result, e = s.Pool.Exec(ctx, q, id)
	}
	if e != nil {
		return e
	}
	if result.RowsAffected() == 0 {
		return domain.Fail(404, "NOT_FOUND", "The record was not found or is not accessible.")
	}
	return nil
}
