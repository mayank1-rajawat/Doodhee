package application_test

import (
	"context"
	"doodhee-backend/internal/adapters/postgres"
	"doodhee-backend/internal/adapters/security"
	"doodhee-backend/internal/application"
	"doodhee-backend/internal/domain"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"
	"os"
	"testing"
	"time"
)

type captureOTP struct{ code string }

func (c *captureOTP) Send(ctx context.Context, phone, code string) error { c.code = code; return nil }
func TestAuthenticationLifecycle(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	p, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	capture := &captureOTP{}
	sec := &security.Security{Key: []byte("authentication-test-key-at-least-32-characters")}
	s := &application.Service{DB: &postgres.Store{Pool: p}, Sec: sec, Sender: capture}
	phone := fmt.Sprintf("+1%d", time.Now().UnixNano()%100000000000)
	ch, e := s.OTPRequest(ctx, "doodhee-public", domain.Record{"phone": phone, "role": "customer"})
	if e != nil {
		t.Fatal(e)
	}
	tokens, e := s.OTPVerify(ctx, "doodhee-public", domain.Record{"challenge_id": ch.Text("challenge_id"), "code": capture.code, "full_name": "Auth test"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.OTPVerify(ctx, "doodhee-public", domain.Record{"challenge_id": ch.Text("challenge_id"), "code": capture.code}); e == nil {
		t.Fatal("OTP replay accepted")
	}
	a, e := s.Authenticate(ctx, tokens.Text("access_token"), "doodhee-public")
	if e != nil {
		t.Fatal(e)
	}
	rotated, e := s.Refresh(ctx, "doodhee-public", domain.Record{"refresh_token": tokens.Text("refresh_token")})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Refresh(ctx, "doodhee-public", domain.Record{"refresh_token": tokens.Text("refresh_token")}); e == nil {
		t.Fatal("refresh reuse accepted")
	}
	if _, e = s.Authenticate(ctx, rotated.Text("access_token"), "doodhee-public"); e == nil {
		t.Fatal("family not revoked")
	}
	if _, e = s.Authenticate(ctx, tokens.Text("access_token"), "doodhee-admin"); e == nil {
		t.Fatal("cross audience accepted")
	}
	_ = a
	secret := "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP"
	adminPhone := fmt.Sprintf("+2%d", time.Now().UnixNano()%100000000000)
	if e = s.SeedAdmin(ctx, adminPhone, "Test admin", secret); e != nil {
		t.Fatal(e)
	}
	ch, e = s.OTPRequest(ctx, "doodhee-admin", domain.Record{"phone": adminPhone})
	if e != nil {
		t.Fatal(e)
	}
	m, e := s.OTPVerify(ctx, "doodhee-admin", domain.Record{"challenge_id": ch.Text("challenge_id"), "code": capture.code})
	if e != nil {
		t.Fatal(e)
	}
	if m["access_token"] != nil {
		t.Fatal("token issued before MFA")
	}
	code, _ := totp.GenerateCode(secret, time.Now())
	adminTokens, e := s.MFA(ctx, domain.Record{"challenge_id": m.Text("mfa_challenge_id"), "code": code})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Authenticate(ctx, adminTokens.Text("access_token"), "doodhee-admin"); e != nil {
		t.Fatal(e)
	}
	if e = s.Logout(ctx, domain.Actor{ID: adminTokens["user"].(domain.Record).Text("id"), Session: func() string { a, _ := sec.Verify(adminTokens.Text("access_token"), "doodhee-admin"); return a.Session }()}, false); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Authenticate(ctx, adminTokens.Text("access_token"), "doodhee-admin"); e == nil {
		t.Fatal("logout did not revoke")
	}
}
