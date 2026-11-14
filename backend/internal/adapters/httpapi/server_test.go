package httpapi

import (
	"doodhee-backend/internal/adapters/security"
	"doodhee-backend/internal/application"
	"doodhee-backend/internal/domain"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGatewayIsolation(t *testing.T) {
	sec := &security.Security{Key: []byte("test-security-key-at-least-32-characters")}
	svc := &application.Service{Sec: sec}
	public := (&Server{Service: svc, Audience: "doodhee-public"}).Handler()
	admin := (&Server{Service: svc, Audience: "doodhee-admin"}).Handler()
	for _, tt := range []struct {
		aud, path string
		admin     bool
	}{{"doodhee-public", "/v1/admin/me", true}, {"doodhee-admin", "/v1/me", false}} {
		token, _ := sec.Token(domain.Actor{ID: sec.ID(), Session: sec.ID(), Role: "customer", Audience: tt.aud})
		r := httptest.NewRequest("GET", tt.path, nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		h := public
		if tt.admin {
			h = admin
		}
		h.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatalf("wrong audience accepted: %d", w.Code)
		}
	}
	r := httptest.NewRequest("GET", "/v1/admin/users", nil)
	w := httptest.NewRecorder()
	public.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatal("admin route registered publicly")
	}
	r = httptest.NewRequest("POST", "/v1/orders", strings.NewReader(`{"customer_id":"injected"}`))
	w = httptest.NewRecorder()
	public.ServeHTTP(w, r)
	if w.Code != 422 {
		t.Fatal("untrusted input accepted")
	}
}
