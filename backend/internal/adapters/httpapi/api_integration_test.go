package httpapi_test

import (
	"bytes"
	"context"
	"doodhee-backend/internal/adapters/httpapi"
	"doodhee-backend/internal/adapters/postgres"
	"doodhee-backend/internal/adapters/security"
	"doodhee-backend/internal/application"
	"doodhee-backend/internal/domain"
	"doodhee-backend/migrations"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"
)

// Capture only this test's OTPs; never read the running application's OTP log.
type testOTP struct {
	mu    sync.Mutex
	codes map[string]string
}

func (o *testOTP) Send(_ context.Context, phone, code string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.codes[phone] = code
	return nil
}
func (o *testOTP) code(phone string) string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.codes[phone]
}

type apiAccount struct{ id, access, refresh, phone string }
type apiReply struct {
	status int
	data   domain.Record
	code   string
	header http.Header
}
type endpointCheck struct {
	Name, Gateway, Method, Path, Role, ErrorCode string
	Status, Expected                             int
	Passed                                       bool
}
type apiSuite struct {
	t             *testing.T
	ctx           context.Context
	svc           *application.Service
	pool          *pgxpool.Pool
	otp           *testOTP
	public, admin *httptest.Server
	client        *http.Client
	checks        []endpointCheck
	assertions    int
}

func newAPISuite(t *testing.T, showOTP ...bool) *apiSuite {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required for real HTTP/PostgreSQL API verification")
	}
	ctx := context.Background()
	control, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	sec := &security.Security{Key: []byte("isolated-api-test-signing-key-32-characters")}
	schema := "api_verify_" + strings.ReplaceAll(sec.ID(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = control.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		control.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := control.Exec(context.Background(), "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
		control.Close()
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	files, err := fs.Glob(migrations.Files, "*.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	for _, file := range files {
		sql, readErr := migrations.Files.ReadFile(file)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, err = pool.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
	}
	otp := &testOTP{codes: map[string]string{}}
	svc := &application.Service{DB: &postgres.Store{Pool: pool}, Sec: sec, Sender: otp, TestingMode: len(showOTP) > 0 && showOTP[0]}
	s := &apiSuite{t: t, ctx: ctx, svc: svc, pool: pool, otp: otp, client: &http.Client{Timeout: 10 * time.Second}}
	s.public = httptest.NewServer((&httpapi.Server{Service: svc, Audience: "doodhee-public", Origins: []string{"http://localhost:3000"}, MaxRequestsPerMinute: 1000}).Handler())
	s.admin = httptest.NewServer((&httpapi.Server{Service: svc, Audience: "doodhee-admin", Origins: []string{"http://localhost:3000"}, MaxRequestsPerMinute: 1000}).Handler())
	t.Cleanup(s.public.Close)
	t.Cleanup(s.admin.Close)
	return s
}

func routeTemplate(gateway, method, path string) string {
	routes := httpapi.Public
	if gateway == "admin" {
		routes = httpapi.Admin
	}
	actual := strings.Split(strings.Split(path, "?")[0], "/")
	for _, r := range routes {
		want := strings.Split(r.Path, "/")
		if method != r.Method || len(actual) != len(want) {
			continue
		}
		match := true
		for i := range want {
			if want[i] != "{id}" && want[i] != actual[i] {
				match = false
			}
		}
		if match {
			return r.Path
		}
	}
	return strings.Split(path, "?")[0]
}

func (s *apiSuite) exchange(gateway, method, path, token string, body []byte, key string, headers http.Header) (apiReply, error) {
	base := s.public.URL
	if gateway == "admin" {
		base = s.admin.URL
	}
	req, err := http.NewRequest(method, base+path, bytes.NewReader(body))
	if err != nil {
		return apiReply{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	for k, vals := range headers {
		req.Header[k] = vals
	}
	r, err := s.client.Do(req)
	if err != nil {
		return apiReply{}, err
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return apiReply{}, err
	}
	out := apiReply{status: r.StatusCode, header: r.Header}
	if r.StatusCode != http.StatusNoContent && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var envelope struct {
			Data  domain.Record
			Error struct{ Code string }
			Meta  struct {
				RequestID string "json:\"request_id\""
			}
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return out, err
		}
		out.data, out.code = envelope.Data, envelope.Error.Code
		if envelope.Meta.RequestID == "" {
			return out, fmt.Errorf("missing response request_id")
		}
	}
	return out, nil
}
func (s *apiSuite) call(name, gateway, role, method, path, token string, body domain.Record, key string, expected int) apiReply {
	raw, _ := json.Marshal(body)
	return s.raw(name, gateway, role, method, path, token, raw, key, nil, expected)
}
func (s *apiSuite) raw(name, gateway, role, method, path, token string, body []byte, key string, headers http.Header, expected int) apiReply {
	s.t.Helper()
	var reply apiReply
	s.t.Run(name, func(t *testing.T) {
		var err error
		reply, err = s.exchange(gateway, method, path, token, body, key, headers)
		pass := err == nil && reply.status == expected
		if expected >= 400 && expected != 404 && reply.code == "" {
			pass = false
		}
		if expected >= 200 && expected < 300 && expected != 204 && !strings.HasPrefix(path, "/health/") && reply.data == nil {
			pass = false
		}
		s.checks = append(s.checks, endpointCheck{name, gateway, method, routeTemplate(gateway, method, path), role, reply.code, reply.status, expected, pass})
		if !pass {
			t.Errorf("%s %s: got HTTP %d (%s), want %d; transport/envelope error: %v", method, path, reply.status, reply.code, expected, err)
		}
	})
	return reply
}
func (s *apiSuite) expect(name string, pass bool, detail string) {
	s.assertions++
	s.t.Run(name, func(t *testing.T) {
		if !pass {
			t.Error(detail)
		}
	})
}
func recordID(t *testing.T, r apiReply) string {
	t.Helper()
	if id := r.data.Text("id"); id != "" {
		return id
	}
	t.Fatal("Fixture creation failed: no record ID (response tokens/body omitted)")
	return ""
}
func items(r apiReply) []any {
	v, _ := r.data["items"].([]any)
	return v
}
func account(r apiReply, phone string) apiAccount {
	u, _ := r.data["user"].(map[string]any)
	return apiAccount{fmt.Sprint(u["id"]), r.data.Text("access_token"), r.data.Text("refresh_token"), phone}
}
func (s *apiSuite) register(role, phone string, requestedMode ...string) apiAccount {
	mode := "register"
	if len(requestedMode) > 0 {
		mode = requestedMode[0]
	}
	request := domain.Record{"phone": phone, "role": role}
	if mode != "auto" {
		request["mode"] = mode
	}
	ch := s.call("register_"+role+phone, "public", role, "POST", "/v1/auth/otp/request", "", request, "", 200)
	v := s.call("verify_"+role+phone, "public", role, "POST", "/v1/auth/otp/verify", "", domain.Record{"challenge_id": ch.data.Text("challenge_id"), "code": s.otp.code(phone), "full_name": "API test " + role}, "", 200)
	a := account(v, phone)
	if a.access == "" {
		s.t.Fatal("Test login did not return tokens")
	}
	return a
}

func (s *apiSuite) report() {
	type coverage struct {
		Gateway, Method, Path string
		Successes, Rejections int
	}
	matrix := []coverage{}
	for _, group := range []struct {
		name   string
		routes []httpapi.Route
	}{{"public", httpapi.Public}, {"admin", httpapi.Admin}} {
		for _, r := range group.routes {
			row := coverage{Gateway: group.name, Method: r.Method, Path: r.Path}
			for _, c := range s.checks {
				if c.Gateway == group.name && c.Method == r.Method && c.Path == r.Path && c.Passed {
					if c.Status >= 200 && c.Status < 300 {
						row.Successes++
					} else {
						row.Rejections++
					}
				}
			}
			matrix = append(matrix, row)
			if row.Successes == 0 {
				s.t.Errorf("No successful HTTP verification for %s %s", r.Method, r.Path)
			}
		}
	}
	failed := 0
	for _, c := range s.checks {
		if !c.Passed {
			failed++
		}
	}
	if dir := os.Getenv("API_REPORT_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			s.t.Error(err)
			return
		}
		data, _ := json.MarshalIndent(map[string]any{"Endpoints": matrix, "Checks": s.checks, "HTTPChecks": len(s.checks), "FailedHTTPChecks": failed, "AdditionalAssertions": s.assertions, "SuitePassed": !s.t.Failed(), "TestEnvironment": "real HTTP servers, PostgreSQL, isolated temporary schema, captured test OTP provider"}, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "api-results.json"), append(data, '\n'), 0644); err != nil {
			s.t.Error(err)
		}
		var md strings.Builder
		fmt.Fprintf(&md, "# Doodhee API endpoint results\n\n%d endpoints; %d HTTP checks; %d additional business/data assertions; %d failed HTTP checks.\n\n", len(matrix), len(s.checks), s.assertions, failed)
		fmt.Fprintln(&md, "| Gateway | Method | Endpoint | Successful cases | Rejection cases |")
		fmt.Fprintln(&md, "| --- | --- | --- | ---: | ---: |")
		for _, r := range matrix {
			fmt.Fprintf(&md, "| %s | %s | %s | %d | %d |\n", r.Gateway, r.Method, r.Path, r.Successes, r.Rejections)
		}
		fmt.Fprintln(&md, "\n## Failed HTTP checks")
		fmt.Fprintln(&md)
		for _, c := range s.checks {
			if !c.Passed {
				fmt.Fprintf(&md, "- %s: got %d (%s), expected %d.\n", c.Name, c.Status, c.ErrorCode, c.Expected)
			}
		}
		if failed == 0 {
			fmt.Fprintln(&md, "None.")
		}
		if err := os.WriteFile(filepath.Join(dir, "api-endpoints.md"), []byte(md.String()), 0644); err != nil {
			s.t.Error(err)
		}
	}
	s.t.Logf("API verification: %d endpoints, %d HTTP checks, %d additional assertions, %d failed HTTP checks", len(matrix), len(s.checks), s.assertions, failed)
}

func TestAllAPIEndpoints(t *testing.T) {
	s := newAPISuite(t)
	defer s.report()
	pub := func(name, role, method, path string, a apiAccount, b domain.Record, key string, want int) apiReply {
		return s.call(name, "public", role, method, path, a.access, b, key, want)
	}
	for _, g := range []string{"public", "admin"} {
		for _, path := range []string{"/health/live", "/health/ready"} {
			s.call(g+path, g, "anonymous", "GET", path, "", nil, "", 200)
		}
	}
	adminPhone, secret := "+15555550000", "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP"
	if err := s.svc.SeedAdmin(s.ctx, adminPhone, "API test admin", secret); err != nil {
		t.Fatal(err)
	}
	ch := s.call("admin_otp_request", "admin", "admin", "POST", "/v1/admin/auth/otp/request", "", domain.Record{"phone": adminPhone}, "", 200)
	s.call("admin_wrong_otp", "admin", "admin", "POST", "/v1/admin/auth/otp/verify", "", domain.Record{"challenge_id": ch.data.Text("challenge_id"), "code": "wrong"}, "", 401)
	mfa := s.call("admin_otp_verify", "admin", "admin", "POST", "/v1/admin/auth/otp/verify", "", domain.Record{"challenge_id": ch.data.Text("challenge_id"), "code": s.otp.code(adminPhone)}, "", 200)
	s.expect("admin_no_tokens_before_MFA", mfa.data.Text("access_token") == "" && mfa.data.Text("mfa_challenge_id") != "", "Admin must complete MFA before receiving tokens")
	s.call("admin_wrong_mfa", "admin", "admin", "POST", "/v1/admin/auth/mfa/verify", "", domain.Record{"challenge_id": mfa.data.Text("mfa_challenge_id"), "code": "wrong"}, "", 401)
	code, _ := totp.GenerateCode(secret, time.Now())
	login := s.call("admin_mfa_verify", "admin", "admin", "POST", "/v1/admin/auth/mfa/verify", "", domain.Record{"challenge_id": mfa.data.Text("mfa_challenge_id"), "code": code}, "", 200)
	admin := account(login, adminPhone)
	if admin.access == "" {
		t.Fatal("Admin MFA login failed")
	}
	adm := func(name, method, path string, b domain.Record, key string, want int) apiReply {
		return s.call(name, "admin", "admin", method, path, admin.access, b, key, want)
	}
	s.call("mfa_replay_rejected", "admin", "admin", "POST", "/v1/admin/auth/mfa/verify", "", domain.Record{"challenge_id": mfa.data.Text("mfa_challenge_id"), "code": code}, "", 401)
	buyer := s.register("customer", "+15555550001")
	otherBuyer := s.register("customer", "+15555550002")
	seller := s.register("seller", "+15555550003")
	otherSeller := s.register("seller", "+15555550004")
	rejectedSeller := s.register("seller", "+15555550005")
	s.call("otp_request_throttled", "public", "buyer", "POST", "/v1/auth/otp/request", "", domain.Record{"phone": buyer.phone, "role": "customer"}, "", 429)
	challenges, err := s.svc.DB.List(s.ctx, "auth_challenges", domain.Record{"phone": buyer.phone})
	if err != nil || len(challenges) != 1 {
		t.Fatal("Buyer challenge fixture missing")
	}
	s.call("public_otp_replay_denied", "public", "buyer", "POST", "/v1/auth/otp/verify", "", domain.Record{"challenge_id": challenges[0].Text("id"), "code": s.otp.code(buyer.phone)}, "", 401)
	for _, g := range []string{"public", "admin"} {
		path := "/v1/auth/token/refresh"
		if g == "admin" {
			path = "/v1/admin/auth/token/refresh"
		}
		s.call("invalid_refresh_"+g, g, "anonymous", "POST", path, "", domain.Record{"refresh_token": "invalid-refresh-token"}, "", 401)
	}
	s.call("public_refresh_on_admin_denied", "admin", "buyer", "POST", "/v1/admin/auth/token/refresh", "", domain.Record{"refresh_token": buyer.refresh}, "", 401)
	lockPhone := "+15555550010"
	locked := s.call("otp_attempt_fixture", "public", "buyer", "POST", "/v1/auth/otp/request", "", domain.Record{"phone": lockPhone, "role": "customer"}, "", 200)
	for i := 0; i < 5; i++ {
		s.call(fmt.Sprintf("wrong_otp_attempt_%d", i+1), "public", "buyer", "POST", "/v1/auth/otp/verify", "", domain.Record{"challenge_id": locked.data.Text("challenge_id"), "code": "invalid"}, "", 401)
	}
	s.call("otp_locked_after_five_attempts", "public", "buyer", "POST", "/v1/auth/otp/verify", "", domain.Record{"challenge_id": locked.data.Text("challenge_id"), "code": s.otp.code(lockPhone), "full_name": "Must not register"}, "", 401)
	expPhone := "+15555550011"
	expired := s.call("otp_expiry_fixture", "public", "buyer", "POST", "/v1/auth/otp/request", "", domain.Record{"phone": expPhone, "role": "customer"}, "", 200)
	if _, err := s.pool.Exec(s.ctx, "UPDATE auth_challenges SET expires_at=now()-interval '1 second' WHERE id=$1", expired.data.Text("challenge_id")); err != nil {
		t.Fatal(err)
	}
	s.call("expired_otp_denied", "public", "buyer", "POST", "/v1/auth/otp/verify", "", domain.Record{"challenge_id": expired.data.Text("challenge_id"), "code": s.otp.code(expPhone), "full_name": "Must not register"}, "", 401)
	unseededPhone := "+15555550012"
	unseeded := s.call("unseeded_admin_challenge", "admin", "anonymous", "POST", "/v1/admin/auth/otp/request", "", domain.Record{"phone": unseededPhone}, "", 200)
	s.call("no_public_admin_registration", "admin", "anonymous", "POST", "/v1/admin/auth/otp/verify", "", domain.Record{"challenge_id": unseeded.data.Text("challenge_id"), "code": s.otp.code(unseededPhone)}, "", 401)
	reason := domain.Record{"reason": "API verification review"}
	adm("admin_self", "GET", "/v1/admin/me", nil, "", 200)
	pub("buyer_self", "buyer", "GET", "/v1/me", buyer, nil, "", 200)
	pub("buyer_update_profile", "buyer", "PATCH", "/v1/me", buyer, domain.Record{"full_name": "API buyer", "address_line": "API house 123", "city": "Amritsar", "pincode": "143001"}, "", 200)
	s.call("login_unknown_account", "public", "anonymous", "POST", "/v1/auth/otp/request", "", domain.Record{"phone": "+15555550990", "role": "customer", "mode": "login"}, "", 404)
	s.call("register_existing_account", "public", "anonymous", "POST", "/v1/auth/otp/request", "", domain.Record{"phone": buyer.phone, "role": "customer", "mode": "register"}, "", 409)
	s.call("login_wrong_account_type", "public", "anonymous", "POST", "/v1/auth/otp/request", "", domain.Record{"phone": buyer.phone, "role": "seller", "mode": "login"}, "", 403)
	if _, err := s.pool.Exec(s.ctx, "UPDATE auth_challenges SET created_at=now()-interval '31 seconds' WHERE phone=$1", otherBuyer.phone); err != nil {
		t.Fatal(err)
	}
	loginChallenge := s.call("existing_buyer_login_request", "public", "buyer", "POST", "/v1/auth/otp/request", "", domain.Record{"phone": otherBuyer.phone, "role": "customer", "mode": "login"}, "", 200)
	loginWithoutName := s.call("existing_buyer_login_without_name", "public", "buyer", "POST", "/v1/auth/otp/verify", "", domain.Record{"challenge_id": loginChallenge.data.Text("challenge_id"), "code": s.otp.code(otherBuyer.phone)}, "", 200)
	s.expect("login_does_not_require_name", loginWithoutName.data.Text("access_token") != "", "An existing account must sign in without submitting a name")
	homeAddress := pub("create_home_address", "buyer", "POST", "/v1/addresses", buyer, domain.Record{"label": "Home", "address_line": "API house 123, Green Street", "city": "Amritsar", "pincode": "143001"}, "", 201)
	homeAddressID := recordID(t, homeAddress)
	s.expect("first_address_is_default", homeAddress.data.Bool("is_default"), "The first saved address must become the default")
	workAddress := pub("create_work_address", "buyer", "POST", "/v1/addresses", buyer, domain.Record{"label": "Work", "address_line": "API office 45, Market Road", "city": "Amritsar", "pincode": "143001", "is_default": true}, "", 201)
	workAddressID := recordID(t, workAddress)
	addressList := pub("list_saved_addresses", "buyer", "GET", "/v1/addresses", buyer, nil, "", 200)
	s.expect("multiple_addresses_saved", len(items(addressList)) == 2, "Buyer must be able to save multiple addresses")
	pub("update_saved_address", "buyer", "PATCH", "/v1/addresses/"+homeAddressID, buyer, domain.Record{"label": "Family home", "address_line": "API house 123, Green Street", "city": "Amritsar", "pincode": "143001"}, "", 200)
	pub("set_default_address", "buyer", "POST", "/v1/addresses/"+homeAddressID+"/default", buyer, domain.Record{}, "", 200)
	pub("other_buyer_cannot_edit_address", "buyer", "PATCH", "/v1/addresses/"+homeAddressID, otherBuyer, domain.Record{"label": "Stolen", "address_line": "Wrong house 000", "city": "Amritsar", "pincode": "143001"}, "", 404)
	pub("delete_saved_address", "buyer", "DELETE", "/v1/addresses/"+workAddressID, buyer, domain.Record{}, "", 200)
	pub("seller_profile", "seller", "GET", "/v1/seller/profile", seller, nil, "", 200)
	status := pub("seller_approval_status_compatibility", "seller", "GET", "/v1/seller/approval-status", seller, nil, "", 200)
	s.expect("seller_starts_approved", status.data.Text("approval_status") == "approved", "New Milkmen must be usable immediately")
	pub("seller_update_profile", "seller", "PATCH", "/v1/seller/profile", seller, domain.Record{"full_name": "API seller", "address_line": "API dairy 123", "city": "Amritsar", "pincode": "143001"}, "", 200)
	pub("seller_immediate_access", "seller", "GET", "/v1/seller/products", seller, nil, "", 200)
	pub("other_seller_immediate_access", "seller", "GET", "/v1/seller/products", otherSeller, nil, "", 200)
	pub("third_seller_immediate_access", "seller", "GET", "/v1/seller/products", rejectedSeller, nil, "", 200)
	adm("removed_pending_seller_endpoint", "GET", "/v1/admin/sellers/pending", nil, "", 404)
	adm("all_users", "GET", "/v1/admin/users", nil, "", 200)
	adm("user_detail", "GET", "/v1/admin/users/"+buyer.id, nil, "", 200)
	filtered := adm("filter_seller_users", "GET", "/v1/admin/users?role=seller&approval_status=approved", nil, "", 200)
	s.expect("user_filters_work", len(items(filtered)) == 3, "All newly registered Milkman accounts must be approved immediately")
	discovered := pub("discover_sellers", "buyer", "GET", "/v1/sellers?pincode=143001", buyer, nil, "", 200)
	s.expect("discovery_filters_and_privacy", len(items(discovered)) == 1 && !strings.Contains(fmt.Sprint(discovered.data), "phone_verified_at"), "Discovery must filter pincode and avoid private fields")
	pub("seller_discovery_detail", "buyer", "GET", "/v1/sellers/"+seller.id, buyer, nil, "", 200)
	pub("third_seller_discoverable", "buyer", "GET", "/v1/sellers/"+rejectedSeller.id, buyer, nil, "", 200)
	p := pub("create_product", "seller", "POST", "/v1/seller/products", seller, domain.Record{"name": "Cow Milk", "unit": "litre", "price": "65.00", "daily_capacity": "10.000"}, "", 201)
	productID := recordID(t, p)
	productPath := "/v1/seller/products/" + productID
	pub("seller_product_list", "seller", "GET", "/v1/seller/products", seller, nil, "", 200)
	pub("seller_product_detail", "seller", "GET", productPath, seller, nil, "", 200)
	pub("update_product", "seller", "PATCH", productPath, seller, domain.Record{"name": "Fresh Cow Milk", "price": "65.00", "daily_capacity": "10.000"}, "", 200)
	pub("disable_availability", "seller", "POST", productPath+"/availability", seller, domain.Record{"is_available": false}, "", 200)
	pub("unavailable_product_hidden", "buyer", "GET", "/v1/products/"+productID, buyer, nil, "", 404)
	pub("enable_availability", "seller", "POST", productPath+"/availability", seller, domain.Record{"is_available": true}, "", 200)
	pub("discover_products", "buyer", "GET", "/v1/sellers/"+seller.id+"/products", buyer, nil, "", 200)
	pub("buyer_product_detail", "buyer", "GET", "/v1/products/"+productID, buyer, nil, "", 200)
	adm("admin_product_list", "GET", "/v1/admin/products", nil, "", 200)
	adm("admin_product_detail", "GET", "/v1/admin/products/"+productID, nil, "", 200)
	adm("suspend_product", "POST", "/v1/admin/products/"+productID+"/suspend", reason, "", 200)
	pub("suspended_product_hidden", "buyer", "GET", "/v1/products/"+productID, buyer, nil, "", 404)
	pub("seller_cannot_unsuspend", "seller", "POST", productPath+"/availability", seller, domain.Record{"is_available": true}, "", 200)
	pub("admin_suspension_persists", "buyer", "GET", "/v1/products/"+productID, buyer, nil, "", 404)
	adm("activate_product", "POST", "/v1/admin/products/"+productID+"/activate", reason, "", 200)
	existingLookup := pub("milkman_finds_existing_customer", "seller", "GET", "/v1/seller/customers/lookup?phone="+url.QueryEscape(buyer.phone), seller, nil, "", 200)
	s.expect("existing_customer_lookup_details", existingLookup.data.Bool("exists") && existingLookup.data.Text("full_name") == "API buyer" && !existingLookup.data.Bool("already_connected"), "Exact phone lookup must show the existing customer before connection")
	missingLookup := pub("milkman_checks_new_customer_phone", "seller", "GET", "/v1/seller/customers/lookup?phone="+url.QueryEscape("+15555550020"), seller, nil, "", 200)
	s.expect("new_customer_lookup", !missingLookup.data.Bool("exists"), "Unknown customer phone must be reported as available for creation")
	pub("milkman_cannot_lookup_non_customer", "seller", "GET", "/v1/seller/customers/lookup?phone="+url.QueryEscape(otherSeller.phone), seller, nil, "", 409)
	createdCustomer := pub("milkman_adds_customer", "seller", "POST", "/v1/seller/customers", seller, domain.Record{"full_name": "Pre-created customer", "phone": "+15555550020", "delivery_address": "20 Milk Lane, Amritsar"}, "", 201)
	createdCustomerID := recordID(t, createdCustomer)
	s.expect("precreated_customer_unverified", createdCustomer.data["phone_verified_at"] == nil, "A Milkman-created customer must remain unverified until OTP login")
	connectedLookup := pub("milkman_sees_created_customer", "seller", "GET", "/v1/seller/customers/lookup?phone="+url.QueryEscape("+15555550020"), seller, nil, "", 200)
	s.expect("created_customer_lookup_details", connectedLookup.data.Bool("exists") && connectedLookup.data.Bool("already_connected") && connectedLookup.data.Text("id") == createdCustomerID, "Newly created customer must immediately appear as connected")
	pub("milkman_customer_phone_global_unique", "seller", "POST", "/v1/seller/customers", seller, domain.Record{"full_name": "Wrong role", "phone": otherSeller.phone, "delivery_address": "20 Milk Lane, Amritsar"}, "", 409)
	customerSearch := pub("search_milkman_customers", "seller", "GET", "/v1/seller/customers?search=50020", seller, nil, "", 200)
	s.expect("customer_search_matches_phone", len(items(customerSearch)) == 1, "Milkman customer search must match phone, name or address")
	pub("milkman_customer_detail", "seller", "GET", "/v1/seller/customers/"+createdCustomerID, seller, nil, "", 200)
	emptyOtherCustomers := pub("other_milkman_customers_are_isolated", "seller", "GET", "/v1/seller/customers?search=50020", otherSeller, nil, "", 200)
	s.expect("milkman_customer_ownership", len(items(emptyOtherCustomers)) == 0, "A Milkman must only see linked customers")
	claimChallenge := s.call("precreated_customer_first_registration_request", "public", "buyer", "POST", "/v1/auth/otp/request", "", domain.Record{"phone": "+15555550020", "role": "customer", "mode": "register"}, "", 200)
	claimedLogin := s.call("precreated_customer_claims_account", "public", "buyer", "POST", "/v1/auth/otp/verify", "", domain.Record{"challenge_id": claimChallenge.data.Text("challenge_id"), "code": s.otp.code("+15555550020"), "full_name": "First app registration"}, "", 200)
	claimedCustomer := account(claimedLogin, "+15555550020")
	s.expect("customer_claim_keeps_identity", claimedCustomer.id == createdCustomerID && claimedCustomer.id != "" && claimedLogin.data.Text("access_token") != "", "First OTP registration or login must claim the exact customer record created by the Milkman")
	pub("link_existing_buyer_to_milkman", "seller", "POST", "/v1/seller/customers", seller, domain.Record{"full_name": "Ignored existing name", "phone": buyer.phone, "delivery_address": "API house, test street"}, "", 201)
	date := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	orderBody := domain.Record{"product_id": productID, "service_date": date, "quantity": "1.500", "delivery_address": "API house, test street", "customer_note": "API test"}
	o := pub("create_order", "buyer", "POST", "/v1/orders", buyer, orderBody, "api-create-key-one", 201)
	orderID := recordID(t, o)
	s.expect("exact_order_total", o.data.Text("total_amount") == "97.50" && o.data.Text("unit_price") == "65.00" && o.data.Text("price_source") == "default", "Default price must be snapshotted when no override exists")
	repeat := pub("order_idempotency_replay", "buyer", "POST", "/v1/orders", buyer, orderBody, "api-create-key-one", 201)
	s.expect("no_duplicate_order", repeat.data.Text("id") == orderID, "Same idempotency key and body must replay original order")
	changed := domain.Record{}
	for k, v := range orderBody {
		changed[k] = v
	}
	changed["quantity"] = "2.000"
	pub("order_idempotency_conflict", "buyer", "POST", "/v1/orders", buyer, changed, "api-create-key-one", 409)
	pub("order_idempotency_required", "buyer", "POST", "/v1/orders", buyer, orderBody, "", 422)
	priceOverride := pub("set_customer_product_price", "seller", "PUT", "/v1/seller/customers/"+buyer.id+"/price", seller, domain.Record{"product_id": productID, "price": "60.00"}, "", 200)
	s.expect("customer_price_saved", priceOverride.data.Text("price") == "60.00", "Customer-specific product price must be stored")
	pub("list_customer_prices", "seller", "GET", "/v1/seller/customers/"+buyer.id+"/prices", seller, nil, "", 200)
	effectiveProduct := pub("buyer_sees_effective_customer_price", "buyer", "GET", "/v1/products/"+productID, buyer, nil, "", 200)
	s.expect("customer_price_precedence", effectiveProduct.data.Text("effective_price") == "60.00" && effectiveProduct.data.Text("price_source") == "customer", "Customer price must override the product default")
	pricedOrderBody := domain.Record{"product_id": productID, "service_date": date, "quantity": "0.500", "delivery_address": "API house, test street", "customer_note": "Customer price snapshot"}
	pricedOrder := pub("create_customer_priced_order", "buyer", "POST", "/v1/orders", buyer, pricedOrderBody, "api-customer-price-order", 201)
	pricedOrderID := recordID(t, pricedOrder)
	s.expect("customer_price_snapshotted", pricedOrder.data.Text("unit_price") == "60.00" && pricedOrder.data.Text("total_amount") == "30.00" && pricedOrder.data.Text("price_source") == "customer", "Customer price must be snapshotted on the delivery")
	rejectedExtra := pub("request_extra_milk_for_rejection", "buyer", "POST", "/v1/orders/"+pricedOrderID+"/extra-milk-request", buyer, domain.Record{"extra_quantity": "0.250"}, "api-extra-reject-create", 201)
	rejectedExtraID := recordID(t, rejectedExtra)
	extraList := pub("seller_lists_extra_requests", "seller", "GET", "/v1/seller/extra-milk-requests?status=pending", seller, nil, "", 200)
	extraItems := items(extraList)
	s.expect("extra_request_has_delivery_context", len(extraItems) == 1 && fmt.Sprint(extraItems[0].(map[string]any)["full_name"]) == "API buyer" && fmt.Sprint(extraItems[0].(map[string]any)["service_date"]) == date, "Milkman request list must identify the customer and delivery date")
	pub("other_seller_cannot_reject_extra", "seller", "POST", "/v1/seller/extra-milk-requests/"+rejectedExtraID+"/reject", otherSeller, domain.Record{"decision_note": "Not mine"}, "api-extra-wrong-seller", 404)
	pub("reject_extra_milk_request", "seller", "POST", "/v1/seller/extra-milk-requests/"+rejectedExtraID+"/reject", seller, domain.Record{"decision_note": "Capacity planning"}, "api-extra-reject", 200)
	pub("read_rejected_extra_request", "buyer", "GET", "/v1/orders/"+pricedOrderID+"/extra-milk-request", buyer, nil, "", 200)
	pub("cancel_customer_priced_order", "buyer", "POST", "/v1/orders/"+pricedOrderID+"/cancel", buyer, reason, "", 200)
	pub("remove_customer_product_price", "seller", "DELETE", "/v1/seller/customers/"+buyer.id+"/price", seller, domain.Record{"product_id": productID}, "", 200)
	historicalPrice := pub("historical_customer_price_unchanged", "buyer", "GET", "/v1/orders/"+pricedOrderID, buyer, nil, "", 200)
	s.expect("historical_price_immutable", historicalPrice.data.Text("unit_price") == "60.00" && historicalPrice.data.Text("total_amount") == "30.00", "Removing a customer override must not rewrite an existing delivery price")
	pendingExtra := pub("request_extra_milk", "buyer", "POST", "/v1/orders/"+orderID+"/extra-milk-request", buyer, domain.Record{"extra_quantity": "0.500"}, "api-extra-create", 201)
	pendingExtraID := recordID(t, pendingExtra)
	pub("read_pending_extra_as_buyer", "buyer", "GET", "/v1/orders/"+orderID+"/extra-milk-request", buyer, nil, "", 200)
	pub("read_pending_extra_as_milkman", "seller", "GET", "/v1/orders/"+orderID+"/extra-milk-request", seller, nil, "", 200)
	approvedExtra := pub("approve_extra_milk_request", "seller", "POST", "/v1/seller/extra-milk-requests/"+pendingExtraID+"/approve", seller, domain.Record{"decision_note": "Added to route"}, "api-extra-approve", 200)
	s.expect("extra_request_approved", approvedExtra.data.Text("status") == "approved", "Milkman must be able to approve an owned extra request")
	plannedOrder := pub("approved_extra_updates_order", "buyer", "GET", "/v1/orders/"+orderID, buyer, nil, "", 200)
	s.expect("approved_extra_quantity_added", plannedOrder.data.Text("quantity") == "2.000" && plannedOrder.data.Text("base_quantity") == "1.500" && plannedOrder.data.Text("approved_extra_quantity") == "0.500", "Approved extra milk must update only the planned delivery quantity")
	pub("second_open_extra_request_rejected", "buyer", "POST", "/v1/orders/"+orderID+"/extra-milk-request", buyer, domain.Record{"extra_quantity": "0.250"}, "api-extra-duplicate", 409)
	firstRating := pub("rate_milkman", "buyer", "PUT", "/v1/sellers/"+seller.id+"/rating", buyer, domain.Record{"rating": "4"}, "", 200)
	ratingID := recordID(t, firstRating)
	updatedRating := pub("edit_milkman_rating", "buyer", "PUT", "/v1/sellers/"+seller.id+"/rating", buyer, domain.Record{"rating": "5"}, "", 200)
	s.expect("rating_edit_reuses_record", updatedRating.data.Text("id") == ratingID && updatedRating.data.Text("rating") == "5", "A customer must edit the one existing Milkman rating")
	pub("read_my_milkman_rating", "buyer", "GET", "/v1/sellers/"+seller.id+"/rating", buyer, nil, "", 200)
	rankings := pub("milkman_rankings", "buyer", "GET", "/v1/sellers/rankings", buyer, nil, "", 200)
	s.expect("rating_updates_ranking", len(items(rankings)) == 3 && fmt.Sprint(items(rankings)[0].(map[string]any)["id"]) == seller.id, "Rated Milkman must rank ahead of unrated Milkmen")
	today := time.Now().In(time.FixedZone("India", 19800)).Format("2006-01-02")
	directOne := pub("record_direct_daily_delivery", "seller", "POST", "/v1/seller/deliveries", seller, domain.Record{"customer_id": createdCustomerID, "product_id": productID, "service_date": today, "quantity": "1.000", "unit_price": "75.00", "note": "Morning delivery"}, "api-direct-delivery-one", 201)
	directTwo := pub("record_second_daily_delivery", "seller", "POST", "/v1/seller/deliveries", seller, domain.Record{"customer_id": createdCustomerID, "product_id": productID, "service_date": today, "quantity": "0.250", "note": "Evening top-up"}, "api-direct-delivery-two", 201)
	s.expect("daily_deliveries_are_separate", directOne.data.Text("id") != directTwo.data.Text("id") && directOne.data.Text("price_source") == "delivery" && directTwo.data.Text("price_source") == "default", "Every daily delivery must be its own order transaction with resolved price source")
	claimedBill := pub("claimed_customer_actual_bill", "buyer", "GET", "/v1/bill-summary?from="+today+"&to="+today, claimedCustomer, nil, "", 200)
	s.expect("direct_delivery_bill", claimedBill.data.Text("delivered_litres") == "1.250" && claimedBill.data.Text("total_amount") == "91.25", "Direct delivery bill must use actual quantities and snapshotted prices")
	pub("buyer_orders", "buyer", "GET", "/v1/orders?service_date="+date, buyer, nil, "", 200)
	pub("buyer_order_detail", "buyer", "GET", "/v1/orders/"+orderID, buyer, nil, "", 200)
	pub("seller_orders", "seller", "GET", "/v1/seller/orders?customer_id="+buyer.id, seller, nil, "", 200)
	pub("seller_order_detail", "seller", "GET", "/v1/seller/orders/"+orderID, seller, nil, "", 200)
	adm("admin_orders", "GET", "/v1/admin/orders?status=pending", nil, "", 200)
	adm("admin_order_detail", "GET", "/v1/admin/orders/"+orderID, nil, "", 200)
	// Missing authentication, wrong roles and gateway separation for every protected route.
	payloads := map[string]domain.Record{
		"profile":                      {"city": "Test"},
		"address.create":               {"label": "Home", "address_line": "Test house 123", "city": "Test city", "pincode": "123456"},
		"address.update":               {"label": "Home", "address_line": "Test house 123", "city": "Test city", "pincode": "123456"},
		"address.delete":               {},
		"address.default":              {},
		"seller.profile.update":        {"city": "Test"},
		"seller.product.create":        {"name": "Milk", "price": "65.00", "daily_capacity": "10.000"},
		"seller.product.update":        {"name": "Milk"},
		"seller.product.availability":  {"is_available": true},
		"seller.customer.create":       {"full_name": "Role test", "phone": "+15555550991", "delivery_address": "Role test address"},
		"seller.customer.price.set":    {"product_id": productID, "price": "60.00"},
		"seller.customer.price.delete": {"product_id": productID},
		"seller.delivery.create":       {"customer_id": buyer.id, "product_id": productID, "service_date": today, "quantity": "1.000"},
		"order.create":                 orderBody,
		"order.cancel":                 {"reason": "API cancel"},
		"extra.request.create":         {"extra_quantity": "0.250"},
		"rating.set":                   {"rating": "5"},
		"seller.order.result":          {"result": "delivered"},
		"seller.extra.approve":         {},
		"seller.extra.reject":          {},
		"admin.order.correct":          {"result": "delivered", "reason": "API correction"},
	}
	pathFor := func(r httpapi.Route) string {
		id := orderID
		if strings.Contains(r.Path, "products/{id}") {
			id = productID
		}
		if strings.Contains(r.Path, "sellers/{id}") || strings.Contains(r.Path, "users/{id}") {
			id = seller.id
		}
		if strings.Contains(r.Path, "addresses/{id}") {
			id = homeAddressID
		}
		return strings.ReplaceAll(r.Path, "{id}", id)
	}
	for _, group := range []struct {
		name   string
		routes []httpapi.Route
	}{{"public", httpapi.Public}, {"admin", httpapi.Admin}} {
		for _, r := range group.routes {
			if strings.HasPrefix(r.Op, "auth.") {
				continue
			}
			body := payloads[r.Op]
			if body == nil {
				body = reason
			}
			if r.Method == "GET" || strings.HasPrefix(r.Op, "logout") || strings.Contains(r.Op, "accept") || strings.Contains(r.Op, "dispatch") {
				body = domain.Record{}
			}
			path := pathFor(r)
			s.call("anonymous_"+group.name+"_"+r.Op, group.name, "anonymous", r.Method, path, "", body, "api-role-test-key", 401)
			if group.name == "admin" {
				for _, a := range []apiAccount{buyer, seller} {
					s.call("public_token_denied_"+r.Op+"_"+a.id, "admin", "buyer/seller", r.Method, path, a.access, body, "api-role-test-key", 401)
				}
			} else {
				s.call("admin_token_denied_"+r.Op, "public", "admin", r.Method, path, admin.access, body, "api-role-test-key", 401)
				if strings.HasPrefix(r.Op, "seller.") {
					s.call("buyer_denied_"+r.Op, "public", "buyer", r.Method, path, buyer.access, body, "api-role-test-key", 403)
				} else if r.Op != "me" && r.Op != "profile" && r.Op != "extra.request.get" && !strings.HasPrefix(r.Op, "logout") {
					s.call("seller_denied_"+r.Op, "public", "seller", r.Method, path, seller.access, body, "api-role-test-key", 403)
				}
			}
		}
	}
	for _, r := range httpapi.Admin {
		s.call("admin_path_not_public_"+r.Op, "public", "admin", r.Method, pathFor(r), admin.access, domain.Record{}, "", 404)
	}
	for _, r := range httpapi.Public {
		s.call("public_path_not_admin_"+r.Op, "admin", "admin", r.Method, pathFor(r), admin.access, domain.Record{}, "", 404)
	}
	// Ownership applies to reads, updates, actions, lists and totals.
	pub("other_buyer_order_hidden", "buyer", "GET", "/v1/orders/"+orderID, otherBuyer, nil, "", 404)
	pub("other_buyer_cannot_cancel", "buyer", "POST", "/v1/orders/"+orderID+"/cancel", otherBuyer, reason, "", 404)
	for _, pair := range []struct {
		method, path string
		body         domain.Record
	}{{"GET", productPath, nil}, {"PATCH", productPath, domain.Record{"name": "Stolen"}}, {"POST", productPath + "/availability", domain.Record{"is_available": false}}, {"GET", "/v1/seller/orders/" + orderID, nil}, {"POST", "/v1/seller/orders/" + orderID + "/accept", domain.Record{}}, {"POST", "/v1/seller/orders/" + orderID + "/out-for-delivery", domain.Record{}}, {"POST", "/v1/seller/orders/" + orderID + "/delivery-result", domain.Record{"result": "delivered"}}} {
		pub("other_seller_"+pair.method+pair.path, "seller", pair.method, pair.path, otherSeller, pair.body, "api-other-seller-key", 404)
	}
	for _, pair := range []struct {
		path string
		a    apiAccount
		role string
	}{{"/v1/orders", otherBuyer, "buyer"}, {"/v1/seller/orders", otherSeller, "seller"}, {"/v1/seller/products", otherSeller, "seller"}} {
		r := pub("other_account_empty_"+pair.path, pair.role, "GET", pair.path, pair.a, nil, "", 200)
		s.expect("ownership_list_"+pair.path, len(items(r)) == 0, "Another account must not see these records")
	}
	pub("cannot_dispatch_pending", "seller", "POST", "/v1/seller/orders/"+orderID+"/out-for-delivery", seller, domain.Record{}, "", 409)
	pub("cannot_mark_pending_delivered", "seller", "POST", "/v1/seller/orders/"+orderID+"/delivery-result", seller, domain.Record{"result": "delivered"}, "api-early-result-key", 409)
	pub("accept_order", "seller", "POST", "/v1/seller/orders/"+orderID+"/accept", seller, domain.Record{}, "", 200)
	pub("dispatch_order", "seller", "POST", "/v1/seller/orders/"+orderID+"/out-for-delivery", seller, domain.Record{}, "", 200)
	pub("missing_delivery_reason", "seller", "POST", "/v1/seller/orders/"+orderID+"/delivery-result", seller, domain.Record{"result": "not_delivered"}, "api-no-reason-key", 422)
	deliveryResult := domain.Record{"result": "delivered", "quantity": "1.800", "unit_price": "70.00", "note": "Delivered to buyer"}
	delivered := pub("mark_delivered", "seller", "POST", "/v1/seller/orders/"+orderID+"/delivery-result", seller, deliveryResult, "api-result-key-one", 200)
	s.expect("delivery_records_actor", delivered.data.Text("marked_by") == seller.id && delivered.data.Text("delivery_marked_at") != "", "Delivery must record seller and timestamp")
	s.expect("actual_delivery_snapshot", delivered.data.Text("quantity") == "1.800" && delivered.data.Text("unit_price") == "70.00" && delivered.data.Text("total_amount") == "126.00" && delivered.data.Text("price_source") == "delivery", "Final delivery must snapshot actual quantity and delivery-specific price")
	pub("delivery_idempotency_replay", "seller", "POST", "/v1/seller/orders/"+orderID+"/delivery-result", seller, deliveryResult, "api-result-key-one", 200)
	for _, pair := range []struct {
		path string
		a    apiAccount
		role string
	}{{"/v1/bill-summary", buyer, "buyer"}, {"/v1/seller/bill-summary", seller, "seller"}} {
		r := pub("matching_summary_"+pair.role, pair.role, "GET", pair.path+"?from="+date+"&to="+date, pair.a, nil, "", 200)
		s.expect("summary_amount_"+pair.role, r.data.Text("total_amount") == "126.00" && r.data.Text("delivered_litres") == "1.800", "Delivered buyer/seller totals must use actual quantity and match")
	}
	adminSum := adm("admin_bill_summary", "GET", "/v1/admin/bill-summary?from="+date+"&to="+date, nil, "", 200)
	s.expect("admin_total_matches", adminSum.data.Text("total_amount") == "126.00", "Admin total must match delivered amount")
	pub("cannot_cancel_delivered", "buyer", "POST", "/v1/orders/"+orderID+"/cancel", buyer, reason, "", 409)
	adm("correction_reason_required", "POST", "/v1/admin/orders/"+orderID+"/correct-delivery-result", domain.Record{"result": "not_delivered", "reason_code": "other", "reason": ""}, "api-correction-empty", 422)
	adm("correct_delivery", "POST", "/v1/admin/orders/"+orderID+"/correct-delivery-result", domain.Record{"result": "not_delivered", "reason_code": "other", "reason": "Buyer confirmed delivery issue"}, "api-correction-key", 200)
	adm("correction_idempotency_replay", "POST", "/v1/admin/orders/"+orderID+"/correct-delivery-result", domain.Record{"result": "not_delivered", "reason_code": "other", "reason": "Buyer confirmed delivery issue"}, "api-correction-key", 200)
	after := pub("buyer_sees_correction", "buyer", "GET", "/v1/orders/"+orderID, buyer, nil, "", 200)
	s.expect("correction_visible", after.data.Text("status") == "not_delivered" && after.data.Text("not_delivered_reason") == "other", "Both result and reason must persist")
	for _, pair := range []struct {
		g, path, amount string
		a               apiAccount
	}{{"public", "/v1/bill-summary", "0.00", buyer}, {"public", "/v1/seller/bill-summary", "91.25", seller}, {"admin", "/v1/admin/bill-summary", "91.25", admin}} {
		r := s.call("corrected_total_"+pair.path, pair.g, "owner", "GET", pair.path, pair.a.access, nil, "", 200)
		s.expect("exclude_not_delivered_"+pair.path, r.data.Text("total_amount") == pair.amount, "The corrected delivery must be excluded without changing other customers' direct deliveries")
	}
	cancel := pub("create_cancellable_order", "buyer", "POST", "/v1/orders", buyer, orderBody, "api-cancel-create-key", 201)
	cancelID := recordID(t, cancel)
	pub("cancel_order", "buyer", "POST", "/v1/orders/"+cancelID+"/cancel", buyer, reason, "", 200)
	dashboard := adm("admin_dashboard", "GET", "/v1/admin/dashboard", nil, "", 200)
	s.expect("dashboard_counts", dashboard.data.Text("users") == "7" && dashboard.data.Text("orders") == "5", "Dashboard counts must include Milkman-created customers and separate direct deliveries")
	events := adm("admin_events", "GET", "/v1/admin/events", nil, "", 200)
	if len(items(events)) == 0 {
		t.Fatal("No outbox events created")
	}
	eventID := fmt.Sprint(items(events)[0].(map[string]any)["id"])
	adm("admin_event_detail", "GET", "/v1/admin/events/"+eventID, nil, "", 200)
	if _, err := s.pool.Exec(s.ctx, "UPDATE outbox_events SET status='failed',attempt_count=5,last_error='Injected test failure' WHERE id=$1", eventID); err != nil {
		t.Fatal(err)
	}
	retry := adm("retry_failed_event", "POST", "/v1/admin/events/"+eventID+"/retry", reason, "", 200)
	s.expect("retry_reset", retry.data.Text("status") == "pending" && retry.data.Text("attempt_count") == "0", "Retry must reset failed event for processing")
	adm("retry_pending_event_rejected", "POST", "/v1/admin/events/"+eventID+"/retry", reason, "", 409)
	audits := adm("audit_log", "GET", "/v1/admin/audit-logs?action=admin.order.correct&target_id="+orderID, nil, "", 200)
	s.expect("correction_audited", len(items(audits)) == 1, "Correction must have exactly one audit record")
	if len(items(audits)) == 1 {
		audit := items(audits)[0].(map[string]any)
		before, _ := audit["before_data"].(map[string]any)
		after, _ := audit["after_data"].(map[string]any)
		s.expect("audit_before_and_after", before["status"] == "delivered" && after["status"] == "not_delivered" && audit["reason"] == "Buyer confirmed delivery issue", "Audit must retain actor, reason and before/after delivery result")
		_, err := s.pool.Exec(s.ctx, "DELETE FROM admin_audit_logs WHERE id=$1", audit["id"])
		s.expect("audit_is_immutable", err != nil && strings.Contains(err.Error(), "immutable"), "Database must reject deleting audit records")
	}
	// Validation probes use realistic invalid inputs, not just missing tokens.
	for _, phone := range []string{"invalid", "++++++++++", "1234567890", "+0000000000"} {
		s.call("invalid_phone_"+phone, "public", "anonymous", "POST", "/v1/auth/otp/request", "", domain.Record{"phone": phone, "role": "customer"}, "", 422)
	}
	for i, role := range []string{"customer", "seller"} {
		phone := []string{"9876543210", "8765432109"}[i]
		if _, err := s.svc.DB.Save(s.ctx, "users", domain.Record{"id": s.svc.Sec.ID(), "phone": phone, "full_name": "Legacy API account", "role": role, "approval_status": "approved"}); err != nil {
			t.Fatal(err)
		}
		legacy := s.register(role, phone, "auto")
		me := pub("legacy_numeric_login_"+role, role, "GET", "/v1/me", legacy, nil, "", 200)
		s.expect("legacy_role_preserved_"+role, me.data.Text("role") == role, "Existing numeric accounts must retain their identity and role")
		pub("legacy_logout_"+role, role, "POST", "/v1/auth/logout", legacy, domain.Record{}, "", 204)
	}
	s.call("public_cannot_register_admin", "public", "anonymous", "POST", "/v1/auth/otp/request", "", domain.Record{"phone": "+15555550009", "role": "admin"}, "", 422)
	pub("profile_role_injection", "buyer", "PATCH", "/v1/me", buyer, domain.Record{"role": "admin"}, "", 422)
	pub("profile_empty_name", "buyer", "PATCH", "/v1/me", buyer, domain.Record{"full_name": ""}, "", 422)
	pub("profile_name_too_long", "buyer", "PATCH", "/v1/me", buyer, domain.Record{"full_name": strings.Repeat("X", 121)}, "", 422)
	pub("seller_profile_empty_name", "seller", "PATCH", "/v1/seller/profile", seller, domain.Record{"full_name": "   "}, "", 422)
	pub("profile_pincode_too_long", "buyer", "PATCH", "/v1/me", buyer, domain.Record{"pincode": strings.Repeat("1", 13)}, "", 422)
	pub("unicode_profile_name", "buyer", "PATCH", "/v1/me", buyer, domain.Record{"full_name": strings.Repeat("क", 120)}, "", 200)
	pub("restore_buyer_profile", "buyer", "PATCH", "/v1/me", buyer, domain.Record{"full_name": "API buyer"}, "", 200)
	for _, b := range []domain.Record{{"name": "Milk", "price": "-1", "daily_capacity": "10"}, {"name": "Milk", "price": "1.999", "daily_capacity": "10"}, {"name": strings.Repeat("X", 121), "price": "65", "daily_capacity": "10"}, {"name": "Milk", "price": "65", "daily_capacity": "10", "unit": "kg"}} {
		pub("invalid_product_"+fmt.Sprint(b), "seller", "POST", "/v1/seller/products", seller, b, "", 422)
	}
	pub("availability_wrong_type", "seller", "POST", productPath+"/availability", seller, domain.Record{"is_available": "true"}, "", 422)
	pub("product_update_name_too_long", "seller", "PATCH", productPath, seller, domain.Record{"name": strings.Repeat("X", 121)}, "", 422)
	for _, pair := range []struct{ field, value string }{{"quantity", "0"}, {"quantity", "-1"}, {"quantity", "1.0001"}, {"quantity", "11.000"}, {"service_date", "invalid"}, {"service_date", time.Now().AddDate(0, 0, -1).Format("2006-01-02")}, {"service_date", time.Now().AddDate(0, 0, 91).Format("2006-01-02")}, {"delivery_address", ""}} {
		b := domain.Record{}
		for k, v := range orderBody {
			b[k] = v
		}
		b[pair.field] = pair.value
		want := 422
		if pair.field == "quantity" && pair.value == "11.000" {
			want = 409
		}
		pub("invalid_order_"+pair.field+pair.value, "buyer", "POST", "/v1/orders", buyer, b, s.svc.Sec.ID(), want)
	}
	s.raw("malformed_json", "public", "buyer", "PATCH", "/v1/me", buyer.access, []byte("{broken"), "", nil, 400)
	s.raw("multiple_json_values", "public", "buyer", "PATCH", "/v1/me", buyer.access, []byte("{\"city\":\"Test\"} {\"role\":\"admin\"}"), "", nil, 400)
	s.raw("oversized_json", "public", "buyer", "PATCH", "/v1/me", buyer.access, []byte("{\"city\":\""+strings.Repeat("x", 70000)+"\"}"), "", nil, 400)
	s.raw("untrusted_origin", "admin", "admin", "GET", "/v1/admin/me", admin.access, nil, "", http.Header{"Origin": []string{"https://untrusted.example"}}, 403)
	s.raw("allowed_origin", "admin", "admin", "GET", "/v1/admin/me", admin.access, nil, "", http.Header{"Origin": []string{"http://localhost:3000"}}, 200)
	emptyCancel := pub("create_empty_reason_fixture", "buyer", "POST", "/v1/orders", buyer, orderBody, "api-empty-cancel-create", 201)
	emptyCancelID := recordID(t, emptyCancel)
	invalidCancel := pub("empty_cancellation_reason", "buyer", "POST", "/v1/orders/"+emptyCancelID+"/cancel", buyer, domain.Record{"reason": ""}, "", 422)
	if invalidCancel.status != 200 {
		pub("cancel_reason_fixture_cleanup", "buyer", "POST", "/v1/orders/"+emptyCancelID+"/cancel", buyer, reason, "", 200)
	}
	// A successful not-delivered flow, distinct from the admin correction.
	missed := pub("create_missed_order", "buyer", "POST", "/v1/orders", buyer, orderBody, "api-missed-create", 201)
	missedID := recordID(t, missed)
	pub("accept_missed_order", "seller", "POST", "/v1/seller/orders/"+missedID+"/accept", seller, domain.Record{}, "", 200)
	pub("dispatch_missed_order", "seller", "POST", "/v1/seller/orders/"+missedID+"/out-for-delivery", seller, domain.Record{}, "", 200)
	missedResult := pub("mark_not_delivered", "seller", "POST", "/v1/seller/orders/"+missedID+"/delivery-result", seller, domain.Record{"result": "not_delivered", "reason_code": "customer_unavailable", "note": "No answer at door"}, "api-missed-result", 200)
	s.expect("non_delivery_reason_visible", missedResult.data.Text("not_delivered_reason") == "customer_unavailable", "Non-delivery must retain its reason")
	// Pagination crosses the fixed 50-record page boundary.
	for i := 0; i < 51; i++ {
		if _, err := s.svc.DB.Save(s.ctx, "products", domain.Record{"id": s.svc.Sec.ID(), "seller_id": seller.id, "name": fmt.Sprintf("Page milk %02d", i), "price": "1.00", "daily_capacity": "1.000"}); err != nil {
			t.Fatal(err)
		}
	}
	page1 := pub("products_page_one", "seller", "GET", "/v1/seller/products", seller, nil, "", 200)
	page2 := pub("products_page_two", "seller", "GET", "/v1/seller/products?cursor="+page1.data.Text("next_cursor"), seller, nil, "", 200)
	s.expect("pagination_complete", len(items(page1)) == 50 && len(items(page2)) == 2 && page1.data.Text("next_cursor") != "" && page2.data.Text("next_cursor") == "", "Pagination must cover all 52 seller products once")
	seen := map[string]bool{}
	unique := true
	for _, row := range append(items(page1), items(page2)...) {
		id := fmt.Sprint(row.(map[string]any)["id"])
		unique = unique && !seen[id]
		seen[id] = true
	}
	s.expect("pagination_no_duplicates", unique && len(seen) == 52, "Pages must return all 52 products without duplication")
	// Real concurrent requests: one remaining litre cannot be sold twice.
	concurrentProduct := pub("create_capacity_race_fixture", "seller", "POST", "/v1/seller/products", seller, domain.Record{"name": "Capacity test milk", "price": "50.00", "daily_capacity": "1.000"}, "", 201)
	concurrentID := recordID(t, concurrentProduct)
	raceBody, _ := json.Marshal(domain.Record{"product_id": concurrentID, "service_date": date, "quantity": "1.000", "delivery_address": "API concurrency test house"})
	replies, failures := make([]apiReply, 2), make([]error, 2)
	var wg sync.WaitGroup
	for i, a := range []apiAccount{buyer, otherBuyer} {
		wg.Add(1)
		go func(i int, a apiAccount) {
			defer wg.Done()
			replies[i], failures[i] = s.exchange("public", "POST", "/v1/orders", a.access, raceBody, fmt.Sprintf("api-capacity-race-%d", i), nil)
		}(i, a)
	}
	wg.Wait()
	statuses := []int{replies[0].status, replies[1].status}
	sort.Ints(statuses)
	s.expect("concurrent_capacity_no_overselling", failures[0] == nil && failures[1] == nil && fmt.Sprint(statuses) == "[201 409]", "Concurrent reservations must produce one order and one capacity rejection")
	pub("expand_concurrency_fixture", "seller", "PATCH", "/v1/seller/products/"+concurrentID, seller, domain.Record{"daily_capacity": "10.000"}, "", 200)
	for i := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			replies[i], failures[i] = s.exchange("public", "POST", "/v1/orders", buyer.access, raceBody, "api-concurrent-replay-key", nil)
		}(i)
	}
	wg.Wait()
	s.expect("concurrent_idempotency_no_duplicates", failures[0] == nil && failures[1] == nil && replies[0].status == 201 && replies[1].status == 201 && replies[0].data.Text("id") == replies[1].data.Text("id"), "Concurrent retries with one key must create exactly one order")
	var orderCount int
	if err := s.pool.QueryRow(s.ctx, "SELECT count(*) FROM orders WHERE product_id=$1", concurrentID).Scan(&orderCount); err != nil {
		t.Fatal(err)
	}
	s.expect("concurrency_database_count", orderCount == 2, "Database should contain one reservation and one idempotent order, not duplicates")
	// Refresh rotation/reuse, logout, and account state changes are last so tokens remain usable above.
	refreshed := s.call("public_refresh", "public", "buyer", "POST", "/v1/auth/token/refresh", "", domain.Record{"refresh_token": otherBuyer.refresh}, "", 200)
	rotated := account(refreshed, otherBuyer.phone)
	s.call("refresh_reuse_denied", "public", "buyer", "POST", "/v1/auth/token/refresh", "", domain.Record{"refresh_token": otherBuyer.refresh}, "", 401)
	pub("refresh_family_revoked", "buyer", "GET", "/v1/me", rotated, nil, "", 401)
	pub("public_logout", "buyer", "POST", "/v1/auth/logout", buyer, domain.Record{}, "", 204)
	pub("logout_token_rejected", "buyer", "GET", "/v1/me", buyer, nil, "", 401)
	pub("public_logout_all", "seller", "POST", "/v1/auth/logout-all", otherSeller, domain.Record{}, "", 204)
	pub("logout_all_token_rejected", "seller", "GET", "/v1/me", otherSeller, nil, "", 401)
	adm("disable_seller", "POST", "/v1/admin/users/"+seller.id+"/disable", reason, "", 200)
	pub("disabled_seller_token_rejected", "seller", "GET", "/v1/me", seller, nil, "", 401)
	s.call("disabled_seller_refresh_denied", "public", "seller", "POST", "/v1/auth/token/refresh", "", domain.Record{"refresh_token": seller.refresh}, "", 401)
	adm("enable_seller", "POST", "/v1/admin/users/"+seller.id+"/enable", reason, "", 200)
	adm("cannot_disable_admin", "POST", "/v1/admin/users/"+admin.id+"/disable", reason, "", 403)
	adminRefresh := s.call("admin_refresh", "admin", "admin", "POST", "/v1/admin/auth/token/refresh", "", domain.Record{"refresh_token": admin.refresh}, "", 200)
	admin = account(adminRefresh, admin.phone)
	adm("admin_logout", "POST", "/v1/admin/auth/logout", domain.Record{}, "", 204)
	adm("admin_logout_token_rejected", "GET", "/v1/admin/me", nil, "", 401)
}
