package main

import (
	"context"
	"doodhee-backend/migrations"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"io/fs"
	"os"
	"sort"
	"strings"
)

func main() {
	ctx := context.Background()
	p, e := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if e != nil {
		panic(e)
	}
	defer p.Close()
	down := len(os.Args) > 1 && os.Args[1] == "down"
	if down && os.Getenv("CONFIRM") != "disposable-database" {
		panic("Down migration deletes Doodhee data. Set CONFIRM=disposable-database for a disposable database.")
	}
	tx, e := p.Begin(ctx)
	if e != nil {
		panic(e)
	}
	defer tx.Rollback(ctx)
	// Public gateway, admin gateway, and worker may start together during a
	// deployment. Serialize their migration checks so only one instance applies
	// a schema version while the others wait for the committed result.
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(1846468943)"); e != nil {
		panic(e)
	}
	if _, e = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); e != nil {
		panic(e)
	}
	var usersExist bool
	if e = tx.QueryRow(ctx, "SELECT to_regclass('public.users') IS NOT NULL").Scan(&usersExist); e != nil {
		panic(e)
	}
	var tracked int
	if e = tx.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&tracked); e != nil {
		panic(e)
	}
	// Databases created before version tracking already contain migration 001.
	if usersExist && tracked == 0 {
		if _, e = tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES('001_initial')"); e != nil {
			panic(e)
		}
	}
	pattern := "*.up.sql"
	if down {
		pattern = "*.down.sql"
	}
	files, e := fs.Glob(migrations.Files, pattern)
	if e != nil {
		panic(e)
	}
	sort.Strings(files)
	if down {
		for left, right := 0, len(files)-1; left < right; left, right = left+1, right-1 {
			files[left], files[right] = files[right], files[left]
		}
	}
	applied := 0
	for _, file := range files {
		version := strings.TrimSuffix(strings.TrimSuffix(file, ".up.sql"), ".down.sql")
		var exists bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", version).Scan(&exists); e != nil {
			panic(e)
		}
		if exists != down {
			continue
		}
		sql, readErr := migrations.Files.ReadFile(file)
		if readErr != nil {
			panic(readErr)
		}
		if _, e = tx.Exec(ctx, string(sql)); e != nil {
			panic(e)
		}
		if down {
			_, e = tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version=$1", version)
		} else {
			_, e = tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES($1)", version)
		}
		if e != nil {
			panic(e)
		}
		fmt.Println("Applied", file)
		applied++
	}
	if down {
		if _, e = tx.Exec(ctx, "DROP TABLE schema_migrations"); e != nil {
			panic(e)
		}
	}
	if e = tx.Commit(ctx); e != nil {
		panic(e)
	}
	if applied == 0 {
		fmt.Println("Schema is current.")
	}
}
