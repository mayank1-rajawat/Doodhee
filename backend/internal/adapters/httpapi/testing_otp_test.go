package httpapi_test

import (
	"doodhee-backend/internal/domain"
	"fmt"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestAllAPITestingOTPMode(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("enabled_%t", enabled), func(t *testing.T) {
			s := newAPISuite(t, enabled)
			secret := "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP"
			for i, role := range []string{"customer", "seller", "admin"} {
				phone := fmt.Sprintf("+1555555100%d", i)
				gateway, prefix := "public", "/v1"
				if role == "admin" {
					gateway, prefix = "admin", "/v1/admin"
					if err := s.svc.SeedAdmin(s.ctx, phone, "Testing mode admin", secret); err != nil {
						t.Fatal(err)
					}
				}
				request := s.call("request_"+role, gateway, role, "POST", prefix+"/auth/otp/request", "", domain.Record{"phone": phone, "role": role}, "", 200)
				value, present := request.data["testing_otp"]
				if present != enabled {
					t.Fatal("OTP exposure must follow the server testing flag")
				}
				if request.header.Get("Cache-Control") != "no-store" {
					t.Fatal("OTP responses must not be cached")
				}
				challenge, err := s.svc.DB.Get(s.ctx, "auth_challenges", request.data.Text("challenge_id"), false)
				if err != nil {
					t.Fatal(err)
				}
				if challenge.Text("code_hash") != s.svc.Sec.Hash(s.otp.code(phone)) {
					t.Fatal("Database must retain the OTP hash")
				}
				if !enabled {
					continue
				}
				code, ok := value.(string)
				if !ok || len(code) != 6 || code != s.otp.code(phone) || request.data.Text("expires_in") != "300" {
					t.Fatal("Testing response must contain the actual delivered six-digit OTP and normal expiry")
				}
				verified := s.call("verify_displayed_"+role, gateway, role, "POST", prefix+"/auth/otp/verify", "", domain.Record{"challenge_id": request.data.Text("challenge_id"), "code": code, "full_name": "Testing OTP " + role}, "", 200)
				if _, ok := verified.data["testing_otp"]; ok {
					t.Fatal("OTP must only be exposed in its request response")
				}
				if role != "admin" {
					if verified.data.Text("access_token") == "" {
						t.Fatal("Displayed OTP must complete normal buyer/seller login")
					}
					continue
				}
				if verified.data.Text("access_token") != "" || verified.data.Text("mfa_challenge_id") == "" {
					t.Fatal("Testing OTP must not bypass admin MFA")
				}
				s.call("mfa_still_required", gateway, role, "POST", prefix+"/auth/mfa/verify", "", domain.Record{"challenge_id": verified.data.Text("mfa_challenge_id"), "code": "invalid"}, "", 401)
				mfaCode, _ := totp.GenerateCode(secret, time.Now())
				mfa := s.call("admin_mfa_after_testing_otp", gateway, role, "POST", prefix+"/auth/mfa/verify", "", domain.Record{"challenge_id": verified.data.Text("mfa_challenge_id"), "code": mfaCode}, "", 200)
				if mfa.data.Text("access_token") == "" {
					t.Fatal("Normal authenticator verification must still complete admin login")
				}
			}
		})
	}
}
