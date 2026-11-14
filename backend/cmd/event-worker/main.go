package main

import (
	"context"
	"doodhee-backend/internal/bootstrap"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	svc, p, e := bootstrap.Open(ctx)
	if e != nil {
		panic(e)
	}
	defer p.Close()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			tx, e := p.Begin(ctx)
			if e != nil {
				continue
			}
			var id, typ string
			var raw []byte
			e = tx.QueryRow(ctx, "SELECT id::text,event_type,payload FROM outbox_events WHERE status='pending' AND next_attempt_at<=now() ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1").Scan(&id, &typ, &raw)
			if e != nil {
				tx.Rollback(ctx)
				continue
			}
			var payload map[string]any
			json.Unmarshal(raw, &payload)
			for _, field := range []string{"customer_id", "seller_id"} {
				if user, ok := payload[field].(string); ok {
					_, e = tx.Exec(ctx, "INSERT INTO notifications(id,user_id,event_id,title,body) VALUES($1,$2,$3,$4,$5) ON CONFLICT(user_id,event_id) DO NOTHING", svc.Sec.ID(), user, id, typ, "Your Doodhee order has been updated.")
					if e != nil {
						break
					}
				}
			}
			if e == nil {
				_, e = tx.Exec(ctx, "UPDATE outbox_events SET status='published',published_at=now(),attempt_count=attempt_count+1 WHERE id=$1", id)
			}
			if e == nil {
				e = tx.Commit(ctx)
			} else {
				tx.Rollback(ctx)
				p.Exec(ctx, "UPDATE outbox_events SET attempt_count=attempt_count+1,last_error='Notification processing failed',status=CASE WHEN attempt_count>=4 THEN 'failed' ELSE 'pending' END,next_attempt_at=now()+interval '30 seconds' WHERE id=$1", id)
			}
			if e != nil {
				slog.Error("event processing failed", "event_id", id)
			}
		}
	}
}
