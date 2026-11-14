package bootstrap

import (
	"context"
	"doodhee-backend/internal/adapters/httpapi"
	"doodhee-backend/internal/adapters/postgres"
	"doodhee-backend/internal/adapters/security"
	"doodhee-backend/internal/application"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func testingMode(appEnv, raw string) (bool, error) {
	if raw == "" {
		return false, nil
	}
	enabled, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("TESTING_MODE must be true or false")
	}
	if enabled && appEnv != "development" && appEnv != "test" {
		return false, fmt.Errorf("TESTING_MODE is allowed only in development or test")
	}
	return enabled, nil
}

func Open(ctx context.Context) (*application.Service, *pgxpool.Pool, error) {
	if os.Getenv("APP_ENV") != "development" && os.Getenv("APP_ENV") != "test" && os.Getenv("APP_ENV") != "production" {
		return nil, nil, fmt.Errorf("APP_ENV must explicitly be development, test, or production")
	}
	showOTP, e := testingMode(os.Getenv("APP_ENV"), os.Getenv("TESTING_MODE"))
	if e != nil {
		return nil, nil, e
	}
	key := os.Getenv("AUTH_KEY")
	if len(key) < 32 {
		return nil, nil, fmt.Errorf("AUTH_KEY must contain at least 32 characters")
	}
	if os.Getenv("APP_ENV") == "production" {
		return nil, nil, fmt.Errorf("production OTP provider is not configured: refusing development OTP adapter")
	}
	pool, e := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if e != nil {
		return nil, nil, e
	}
	if e = pool.Ping(ctx); e != nil {
		return nil, nil, e
	}
	return &application.Service{DB: &postgres.Store{Pool: pool}, Sec: &security.Security{Key: []byte(key)}, Sender: security.DevOTP{}, TestingMode: showOTP}, pool, nil
}
func Run(audience, addr string) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	svc, p, e := Open(ctx)
	if e != nil {
		slog.Error("startup failed", "error", e)
		os.Exit(1)
	}
	defer p.Close()
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		addr = v
	}
	keyName := "PUBLIC_HTTP_ADDR"
	if audience == "doodhee-admin" {
		keyName = "ADMIN_HTTP_ADDR"
	}
	if v := os.Getenv(keyName); v != "" {
		addr = v
	}
	handler := (&httpapi.Server{Service: svc, Audience: audience, Origins: strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")}).Handler()
	server := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(c)
	}()
	slog.Info("gateway ready", "audience", audience, "address", addr)
	if e = server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		slog.Error("server failed", "error", e)
		os.Exit(1)
	}
}
