package application_test

import (
	"context"
	"doodhee-backend/internal/adapters/postgres"
	"doodhee-backend/internal/adapters/security"
	"doodhee-backend/internal/application"
	"doodhee-backend/internal/domain"
	"doodhee-backend/internal/ports"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestMarketplace(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	pool, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	sec := &security.Security{Key: []byte("integration-only-32-character-key-000")}
	s := &application.Service{DB: &postgres.Store{Pool: pool}, Sec: sec, Sender: security.DevOTP{}}
	mk := func(role string) domain.Actor {
		u, e := s.DB.Save(ctx, "users", domain.Record{"id": sec.ID(), "phone": sec.ID()[:18], "full_name": "Test " + role, "role": role, "approval_status": "approved"})
		if e != nil {
			t.Fatal(e)
		}
		return domain.Actor{ID: u.Text("id"), Role: role, Approval: "approved", Enabled: true}
	}
	seller, buyer, other, admin := mk("seller"), mk("customer"), mk("seller"), mk("admin")
	run := func(a domain.Actor, op, id string, in domain.Record, key string) any {
		out, e := s.Execute(ctx, a, op, id, in, url.Values{}, "test-request", key)
		if e != nil {
			t.Fatalf("%s: %v", op, e)
		}
		return out
	}
	p := run(seller, "seller.product.create", "", domain.Record{"name": "Cow Milk", "price": "65.00", "daily_capacity": "10.000"}, "").(domain.Record)
	id := p.Text("id")
	input := domain.Record{"product_id": id, "service_date": time.Now().Add(24 * time.Hour).Format("2006-01-02"), "quantity": "1.500", "delivery_address": "Test address 123"}
	o := run(buyer, "order.create", "", input, "create-key-123").(domain.Record)
	if o.Text("total_amount") != "97.50" {
		t.Fatal(o)
	}
	again := run(buyer, "order.create", "", input, "create-key-123").(domain.Record)
	if again.Text("id") != o.Text("id") {
		t.Fatal("duplicate order")
	}
	input["quantity"] = "2.000"
	if _, e = s.Execute(ctx, buyer, "order.create", "", input, nil, "test", "create-key-123"); e == nil {
		t.Fatal("expected idempotency conflict")
	}
	if _, e = s.Execute(ctx, other, "seller.order.accept", o.Text("id"), domain.Record{}, nil, "test", ""); e == nil {
		t.Fatal("ownership failure")
	}
	run(seller, "seller.order.accept", o.Text("id"), domain.Record{}, "")
	run(seller, "seller.order.dispatch", o.Text("id"), domain.Record{}, "")
	if _, e = s.Execute(ctx, seller, "seller.order.result", o.Text("id"), domain.Record{"result": "not_delivered"}, nil, "test", "result-key-1"); e == nil {
		t.Fatal("reason required")
	}
	run(seller, "seller.order.result", o.Text("id"), domain.Record{"result": "delivered"}, "result-key-2")
	sum := run(buyer, "summary", "", nil, "").(domain.Record)
	if sum.Text("total_amount") != "97.50" {
		t.Fatal(sum)
	}
	run(admin, "admin.order.correct", o.Text("id"), domain.Record{"result": "not_delivered", "reason_code": "other", "reason": "Test correction"}, "correct-key-1")
	sum = run(buyer, "summary", "", nil, "").(domain.Record)
	if sum.Text("total_amount") != "0.00" {
		t.Fatal(sum)
	}
	pending := seller
	pending.Approval = "pending"
	if _, e = s.Execute(ctx, pending, "seller.products", "", nil, nil, "", ""); e != nil {
		t.Fatalf("seller approval must not gate access: %v", e)
	}
	token, e := s.Sec.Token(domain.Actor{ID: buyer.ID, Session: sec.ID(), Role: "customer", Audience: "doodhee-public"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Sec.Verify(token, "doodhee-admin"); e == nil {
		t.Fatal("audience isolation")
	}
	// Inject a failure after a write and verify transaction rollback.
	doomed := sec.ID()
	e = s.DB.Tx(ctx, func(db ports.Store) error {
		_, e := db.Save(ctx, "products", domain.Record{"id": doomed, "seller_id": seller.ID, "name": "Rollback", "price": "1", "daily_capacity": "1"})
		if e != nil {
			return e
		}
		return fmt.Errorf("injected failure")
	})
	if e == nil {
		t.Fatal("expected rollback")
	}
	if _, e = s.DB.Get(ctx, "products", doomed, false); e == nil {
		t.Fatal("transaction leaked")
	}
}
