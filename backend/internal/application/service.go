package application

import (
	"context"
	"crypto/subtle"
	"doodhee-backend/internal/domain"
	"doodhee-backend/internal/ports"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	DB          ports.Store
	Sec         ports.Security
	Sender      ports.OTPSender
	TestingMode bool
}

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }
func expired(r domain.Record, key string) bool {
	t, e := time.Parse(time.RFC3339Nano, r.Text(key))
	return e != nil || time.Now().After(t)
}
func validPhone(phone string) bool {
	return len(phone) >= 10 && len(phone) <= 16 && phone[0] == '+' && phone[1] >= '1' && phone[1] <= '9' && strings.Trim(phone[1:], "0123456789") == ""
}
func (s *Service) Event(ctx context.Context, db ports.Store, typ, entity, id string, payload domain.Record) error {
	b, _ := json.Marshal(payload)
	_, e := db.Save(ctx, "outbox_events", domain.Record{"id": s.Sec.ID(), "event_type": typ, "aggregate_type": entity, "aggregate_id": id, "payload": string(b)})
	return e
}
func (s *Service) Audit(ctx context.Context, db ports.Store, a domain.Actor, action, table, id, reason, req string, before, after domain.Record) error {
	b, _ := json.Marshal(before)
	n, _ := json.Marshal(after)
	_, e := db.Save(ctx, "admin_audit_logs", domain.Record{"id": s.Sec.ID(), "admin_id": a.ID, "action": action, "target_type": table, "target_id": id, "reason": reason, "request_id": req, "before_data": string(b), "after_data": string(n)})
	return e
}
func (s *Service) Authenticate(ctx context.Context, raw, aud string) (domain.Actor, error) {
	a, e := s.Sec.Verify(raw, aud)
	if e != nil {
		return a, e
	}
	session, e := s.DB.Get(ctx, "auth_sessions", a.Session, false)
	if e != nil || session.Text("user_id") != a.ID || session.Text("audience") != aud || session["revoked_at"] != nil || expired(session, "expires_at") {
		return a, domain.Fail(401, "AUTH_SESSION_REVOKED", "Your session has ended.")
	}
	u, e := s.DB.Get(ctx, "users", a.ID, false)
	if e != nil {
		return a, e
	}
	a.Role = u.Text("role")
	a.Approval = u.Text("approval_status")
	a.Enabled = u.Bool("is_enabled")
	if !a.Enabled {
		return a, domain.Fail(403, "ACCOUNT_DISABLED", "This account is disabled.")
	}
	if (aud == "doodhee-admin") != (a.Role == "admin") {
		return a, domain.Fail(403, "ROLE_FORBIDDEN", "This gateway is not available for your role.")
	}
	return a, nil
}
func (s *Service) OTPRequest(ctx context.Context, aud string, in domain.Record) (domain.Record, error) {
	phone := in.Text("phone")
	role := in.Text("role")
	mode := in.Text("mode")
	if mode == "" {
		mode = "auto"
	}
	phoneIsValid := validPhone(phone)
	// Preserve login for numeric accounts created before country-code validation.
	// New accounts still require the documented +country-code format.
	if !phoneIsValid && len(phone) >= 10 && len(phone) <= 15 && phone[0] >= '1' && phone[0] <= '9' && strings.Trim(phone, "0123456789") == "" {
		existing, e := s.DB.List(ctx, "users", domain.Record{"phone": phone})
		if e != nil {
			return nil, e
		}
		phoneIsValid = len(existing) > 0
	}
	if !phoneIsValid {
		return nil, domain.Fail(422, "VALIDATION_ERROR", "Enter a valid phone number including country code.")
	}
	if aud == "doodhee-admin" {
		role = "admin"
		mode = "auto"
	}
	if role != "customer" && role != "seller" && !(role == "admin" && aud == "doodhee-admin") {
		return nil, domain.Fail(422, "ROLE_FORBIDDEN", "Choose buyer or seller.")
	}
	if mode != "auto" && mode != "login" && mode != "register" {
		return nil, domain.Fail(422, "VALIDATION_ERROR", "Choose sign in or create account.")
	}
	id := s.Sec.ID()
	code := s.Sec.OTP()
	e := s.DB.Tx(ctx, func(db ports.Store) error {
		if e := db.Lock(ctx, "otp:"+phone); e != nil {
			return e
		}
		if mode != "auto" {
			users, e := db.List(ctx, "users", domain.Record{"phone": phone})
			if e != nil {
				return e
			}
			if mode == "login" {
				if len(users) == 0 {
					return domain.Fail(404, "ACCOUNT_NOT_FOUND", "No account was found for this phone number. Create an account first.")
				}
				if users[0].Text("role") != role {
					return domain.Fail(403, "ACCOUNT_ROLE_MISMATCH", "This phone number is registered with a different account type.")
				}
			}
			if mode == "register" && len(users) > 0 {
				claimableCustomer := users[0].Text("role") == "customer" && role == "customer" && users[0]["phone_verified_at"] == nil
				if !claimableCustomer {
					return domain.Fail(409, "ACCOUNT_EXISTS", "An account already exists for this phone number. Sign in instead.")
				}
			}
		}
		old, e := db.List(ctx, "auth_challenges", domain.Record{"phone": phone})
		if e != nil {
			return e
		}
		recent := 0
		for _, r := range old {
			t, _ := time.Parse(time.RFC3339Nano, r.Text("created_at"))
			if time.Since(t) < 30*time.Second {
				return domain.Fail(429, "RATE_LIMIT", "Wait 30 seconds before requesting another code.")
			}
			if time.Since(t) < time.Hour {
				recent++
			}
		}
		if recent >= 5 {
			return domain.Fail(429, "RATE_LIMIT", "Too many codes requested. Please try later.")
		}
		_, e = db.Save(ctx, "auth_challenges", domain.Record{"id": id, "phone": phone, "role": role, "audience": aud, "purpose": "otp", "auth_mode": mode, "code_hash": s.Sec.Hash(code), "expires_at": time.Now().Add(5 * time.Minute).UTC().Format(time.RFC3339Nano)})
		return e
	})
	if e != nil {
		return nil, e
	}
	if e = s.Sender.Send(ctx, phone, code); e != nil {
		return nil, e
	}
	out := domain.Record{"challenge_id": id, "expires_in": 300}
	if s.TestingMode {
		out["testing_otp"] = code
	}
	return out, nil
}
func (s *Service) session(ctx context.Context, db ports.Store, u domain.Record, aud, family string) (domain.Record, error) {
	raw := s.Sec.Random()
	id := s.Sec.ID()
	if family == "" {
		family = s.Sec.ID()
	}
	ttl := 30 * 24 * time.Hour
	if aud == "doodhee-admin" {
		ttl = 8 * time.Hour
	}
	_, e := db.Save(ctx, "auth_sessions", domain.Record{"id": id, "user_id": u.Text("id"), "audience": aud, "refresh_token_hash": s.Sec.Hash(raw), "token_family_id": family, "expires_at": time.Now().Add(ttl).UTC().Format(time.RFC3339Nano)})
	if e != nil {
		return nil, e
	}
	token, e := s.Sec.Token(domain.Actor{ID: u.Text("id"), Role: u.Text("role"), Session: id, Audience: aud})
	return domain.Record{"access_token": token, "refresh_token": raw, "user": u}, e
}
func (s *Service) OTPVerify(ctx context.Context, aud string, in domain.Record) (domain.Record, error) {
	var out domain.Record
	var denied error
	e := s.DB.Tx(ctx, func(db ports.Store) error {
		r, e := db.Get(ctx, "auth_challenges", in.Text("challenge_id"), true)
		if e != nil {
			return domain.Fail(401, "AUTH_INVALID_OTP", "Invalid or expired code.")
		}
		if r.Text("audience") != aud || r.Text("purpose") != "otp" || r["consumed_at"] != nil || expired(r, "expires_at") || r.Text("attempt_count") == "5" {
			return domain.Fail(401, "AUTH_INVALID_OTP", "Invalid or expired code.")
		}
		attempt := 0
		fmt.Sscan(r.Text("attempt_count"), &attempt)
		r["attempt_count"] = attempt + 1
		if subtle.ConstantTimeCompare([]byte(r.Text("code_hash")), []byte(s.Sec.Hash(in.Text("code")))) != 1 {
			_, e = db.Save(ctx, "auth_challenges", r)
			denied = domain.Fail(401, "AUTH_INVALID_OTP", "Invalid or expired code.")
			return e
		}
		r["consumed_at"] = now()
		if _, e = db.Save(ctx, "auth_challenges", r); e != nil {
			return e
		}
		if e = db.Lock(ctx, "user:"+r.Text("phone")); e != nil {
			return e
		}
		users, e := db.List(ctx, "users", domain.Record{"phone": r.Text("phone")})
		if e != nil {
			return e
		}
		var u domain.Record
		if len(users) == 0 {
			if r.Text("auth_mode") == "login" {
				return domain.Fail(404, "ACCOUNT_NOT_FOUND", "No account was found for this phone number. Create an account first.")
			}
			if aud == "doodhee-admin" {
				return domain.Fail(401, "AUTH_INVALID_OTP", "Unable to sign in.")
			}
			if e := validateName(in.Text("full_name"), 1, "Full name"); e != nil {
				return e
			}
			u, e = db.Save(ctx, "users", domain.Record{"id": s.Sec.ID(), "phone": r.Text("phone"), "full_name": in.Text("full_name"), "role": r.Text("role"), "approval_status": "approved", "phone_verified_at": now()})
			if e != nil {
				return e
			}
			typ := "UserRegistered"
			if u.Text("role") == "seller" {
				typ = "SellerRegistered"
			}
			if e = s.Event(ctx, db, typ, "user", u.Text("id"), u); e != nil {
				return e
			}
		} else {
			u = users[0]
			claimableCustomer := u.Text("role") == "customer" && r.Text("role") == "customer" && u["phone_verified_at"] == nil
			if r.Text("auth_mode") == "register" && !claimableCustomer {
				return domain.Fail(409, "ACCOUNT_EXISTS", "An account already exists for this phone number. Sign in instead.")
			}
			if u["phone_verified_at"] == nil {
				u["phone_verified_at"] = now()
				u["updated_at"] = now()
				u, e = db.Save(ctx, "users", u)
				if e != nil {
					return e
				}
				if e = s.Event(ctx, db, "UserAccountClaimed", "user", u.Text("id"), u); e != nil {
					return e
				}
			}
		}
		if !u.Bool("is_enabled") {
			return domain.Fail(403, "ACCOUNT_DISABLED", "This account is disabled.")
		}
		if (u.Text("role") == "admin") != (aud == "doodhee-admin") {
			return domain.Fail(403, "ROLE_FORBIDDEN", "Use the appropriate application for this account.")
		}
		if aud == "doodhee-admin" {
			id := s.Sec.ID()
			_, e = db.Save(ctx, "auth_challenges", domain.Record{"id": id, "phone": u.Text("phone"), "role": "admin", "audience": aud, "purpose": "mfa", "code_hash": "", "expires_at": time.Now().Add(3 * time.Minute).UTC().Format(time.RFC3339Nano)})
			out = domain.Record{"mfa_challenge_id": id}
			return e
		}
		out, e = s.session(ctx, db, u, aud, "")
		return e
	})
	if denied != nil {
		return nil, denied
	}
	return out, e
}
func (s *Service) MFA(ctx context.Context, in domain.Record) (domain.Record, error) {
	var out domain.Record
	var denied error
	e := s.DB.Tx(ctx, func(db ports.Store) error {
		r, e := db.Get(ctx, "auth_challenges", in.Text("challenge_id"), true)
		if e != nil {
			return e
		}
		attempt := 0
		fmt.Sscan(r.Text("attempt_count"), &attempt)
		if r.Text("purpose") != "mfa" || r["consumed_at"] != nil || expired(r, "expires_at") || attempt >= 5 {
			return domain.Fail(401, "AUTH_INVALID_MFA", "Sign in again.")
		}
		r["attempt_count"] = attempt + 1
		if _, e = db.Save(ctx, "auth_challenges", r); e != nil {
			return e
		}
		users, e := db.List(ctx, "users", domain.Record{"phone": r.Text("phone")})
		if e != nil || len(users) == 0 {
			return domain.Fail(401, "AUTH_INVALID_MFA", "Unable to sign in.")
		}
		u := users[0]
		methods, e := db.List(ctx, "mfa_methods", domain.Record{"user_id": u.Text("id")})
		if e != nil || len(methods) == 0 {
			return domain.Fail(401, "AUTH_INVALID_MFA", "Admin MFA is not configured.")
		}
		m, e := db.Get(ctx, "mfa_methods", methods[0].Text("id"), true)
		if e != nil {
			return e
		}
		secret, e := s.Sec.Decrypt(m.Text("secret_encrypted"))
		if e != nil {
			return e
		}
		step := time.Now().Unix() / 30
		last := int64(0)
		fmt.Sscan(m.Text("last_used_step"), &last)
		if !u.Bool("is_enabled") || m["disabled_at"] != nil || last >= step || !s.Sec.ValidateTOTP(in.Text("code"), secret) {
			denied = domain.Fail(401, "AUTH_INVALID_MFA", "Invalid or already used MFA code.")
			return nil
		}
		m["last_used_step"] = step
		if _, e = db.Save(ctx, "mfa_methods", m); e != nil {
			return e
		}
		r["consumed_at"] = now()
		if _, e = db.Save(ctx, "auth_challenges", r); e != nil {
			return e
		}
		out, e = s.session(ctx, db, u, "doodhee-admin", "")
		return e
	})
	if denied != nil {
		return nil, denied
	}
	return out, e
}
func (s *Service) Refresh(ctx context.Context, aud string, in domain.Record) (domain.Record, error) {
	var out domain.Record
	var denied error
	e := s.DB.Tx(ctx, func(db ports.Store) error {
		rows, e := db.List(ctx, "auth_sessions", domain.Record{"refresh_token_hash": s.Sec.Hash(in.Text("refresh_token"))})
		if e != nil || len(rows) == 0 {
			return domain.Fail(401, "AUTH_SESSION_REVOKED", "Sign in again.")
		}
		r, e := db.Get(ctx, "auth_sessions", rows[0].Text("id"), true)
		if e != nil {
			return e
		}
		if r.Text("audience") != aud {
			return domain.Fail(401, "AUTH_SESSION_REVOKED", "Sign in again.")
		}
		if r["revoked_at"] != nil {
			family, e := db.List(ctx, "auth_sessions", domain.Record{"token_family_id": r.Text("token_family_id")})
			if e != nil {
				return e
			}
			for _, sess := range family {
				sess["revoked_at"] = now()
				if _, e = db.Save(ctx, "auth_sessions", sess); e != nil {
					return e
				}
			}
			denied = domain.Fail(401, "AUTH_SESSION_REVOKED", "Token reuse detected. Sign in again.")
			return nil
		}
		if expired(r, "expires_at") {
			return domain.Fail(401, "AUTH_SESSION_REVOKED", "Sign in again.")
		}
		u, e := db.Get(ctx, "users", r.Text("user_id"), false)
		if e != nil {
			return e
		}
		if !u.Bool("is_enabled") {
			return domain.Fail(403, "ACCOUNT_DISABLED", "This account is disabled.")
		}
		r["revoked_at"] = now()
		if _, e = db.Save(ctx, "auth_sessions", r); e != nil {
			return e
		}
		out, e = s.session(ctx, db, u, aud, r.Text("token_family_id"))
		return e
	})
	if denied != nil {
		return nil, denied
	}
	return out, e
}
func (s *Service) Logout(ctx context.Context, a domain.Actor, all bool) error {
	return s.DB.Tx(ctx, func(db ports.Store) error {
		rows, e := db.List(ctx, "auth_sessions", domain.Record{"user_id": a.ID})
		if e != nil {
			return e
		}
		for _, r := range rows {
			if all || r.Text("id") == a.Session {
				r["revoked_at"] = now()
				if _, e = db.Save(ctx, "auth_sessions", r); e != nil {
					return e
				}
			}
		}
		return nil
	})
}
