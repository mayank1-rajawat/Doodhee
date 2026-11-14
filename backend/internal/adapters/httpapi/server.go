package httpapi

import (
	contract "doodhee-backend/contracts/api"
	"doodhee-backend/internal/application"
	"doodhee-backend/internal/domain"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type bucket struct {
	count int
	reset time.Time
}
type Server struct {
	Service  *application.Service
	Audience string
	Origins  []string
	// MaxRequestsPerMinute defaults to 300. Tests may raise it while exercising
	// the complete endpoint/role matrix through one in-process client address.
	MaxRequestsPerMinute int
	mu                   sync.Mutex
	limits               map[string]bucket
}

func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	m.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		if s.Service.DB.Ping(r.Context()) != nil {
			http.Error(w, "unavailable", 503)
			return
		}
		w.Write([]byte("ok"))
	})
	routes := Public
	if s.Audience == "doodhee-admin" {
		routes = Admin
	}
	for _, route := range routes {
		rt := route
		m.HandleFunc(rt.Method+" "+rt.Path, func(w http.ResponseWriter, r *http.Request) { s.serve(w, r, rt) })
	}
	return m
}
func (s *Server) serve(w http.ResponseWriter, r *http.Request, rt Route) {
	id := r.Header.Get("X-Request-ID")
	if len(id) < 8 || len(id) > 100 || strings.ContainsAny(id, "\r\n") {
		id = s.Service.Sec.ID()
	}
	w.Header().Set("X-Request-ID", id)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	start := time.Now()
	defer func() {
		if recover() != nil {
			json.NewEncoder(w).Encode(domain.Record{"error": domain.Record{"code": "INTERNAL", "message": "Unexpected server error."}})
		}
		slog.Info("request", "request_id", id, "method", r.Method, "route", rt.Path, "duration_ms", time.Since(start).Milliseconds())
	}()
	respond := func(status int, data any, err error) {
		if err != nil {
			var de *domain.Error
			if !errors.As(err, &de) {
				slog.Error("operation failed", "request_id", id, "error", err)
				de = &domain.Error{Status: 500, Code: "INTERNAL", Message: "The operation could not be completed."}
			}
			w.WriteHeader(de.Status)
			json.NewEncoder(w).Encode(domain.Record{"error": domain.Record{"code": de.Code, "message": de.Message, "fields": domain.Record{}}, "meta": domain.Record{"request_id": id}})
			return
		}
		w.WriteHeader(status)
		if status != 204 {
			json.NewEncoder(w).Encode(domain.Record{"data": data, "meta": domain.Record{"request_id": id}})
		}
	}
	origin := r.Header.Get("Origin")
	if origin != "" {
		allowed := false
		for _, o := range s.Origins {
			if origin == o {
				allowed = true
			}
		}
		if !allowed {
			respond(0, nil, domain.Fail(403, "ORIGIN_FORBIDDEN", "Origin not allowed."))
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	s.mu.Lock()
	if s.limits == nil {
		s.limits = map[string]bucket{}
	}
	b := s.limits[ip]
	if time.Now().After(b.reset) {
		b = bucket{reset: time.Now().Add(time.Minute)}
	}
	b.count++
	s.limits[ip] = b
	s.mu.Unlock()
	limit := s.MaxRequestsPerMinute
	if limit == 0 {
		limit = 300
	}
	if b.count > limit {
		w.Header().Set("Retry-After", "60")
		respond(0, nil, domain.Fail(429, "RATE_LIMIT", "Please wait and retry."))
		return
	}
	in := domain.Record{}
	if r.Method != "GET" {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		dec := json.NewDecoder(r.Body)
		dec.UseNumber()
		if e := dec.Decode(&in); e != nil && e != io.EOF {
			respond(0, nil, domain.Fail(400, "INVALID_JSON", "Provide a valid JSON object."))
			return
		}
		if e := dec.Decode(new(any)); e != io.EOF {
			respond(0, nil, domain.Fail(400, "INVALID_JSON", "Provide exactly one JSON object."))
			return
		}
		if in == nil {
			respond(0, nil, domain.Fail(400, "INVALID_JSON", "Provide a JSON object."))
			return
		}
	}
	if r.Method != "GET" {
		valid := map[string]bool{}
		for _, k := range contract.Fields[rt.Op] {
			valid[k] = true
		}
		for k, v := range in {
			if !valid[k] {
				respond(0, nil, domain.Fail(422, "VALIDATION_ERROR", "Unexpected input field: "+k))
				return
			}
			if k == "is_available" || k == "is_default" {
				if _, ok := v.(bool); !ok {
					respond(0, nil, domain.Fail(422, "VALIDATION_ERROR", "Availability must be a boolean."))
					return
				}
			} else {
				if _, ok := v.(string); !ok {
					respond(0, nil, domain.Fail(422, "VALIDATION_ERROR", "Fields must be strings."))
					return
				}
				if len(in.Text(k)) > 2000 {
					respond(0, nil, domain.Fail(422, "VALIDATION_ERROR", "Input is too long."))
					return
				}
			}
		}
		for _, k := range contract.Required[rt.Op] {
			if _, ok := in[k]; !ok {
				respond(0, nil, domain.Fail(422, "VALIDATION_ERROR", "Missing required field: "+k))
				return
			}
		}
	}
	var data any
	var e error
	switch rt.Op {
	case "auth.request":
		data, e = s.Service.OTPRequest(r.Context(), s.Audience, in)
	case "auth.verify":
		data, e = s.Service.OTPVerify(r.Context(), s.Audience, in)
	case "auth.mfa":
		data, e = s.Service.MFA(r.Context(), in)
	case "auth.refresh":
		data, e = s.Service.Refresh(r.Context(), s.Audience, in)
	default:
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		var a domain.Actor
		a, e = s.Service.Authenticate(r.Context(), raw, s.Audience)
		if e == nil {
			if strings.HasPrefix(rt.Op, "logout") {
				e = s.Service.Logout(r.Context(), a, rt.Op == "logout.all")
				if e == nil {
					respond(204, nil, nil)
					return
				}
			} else {
				data, e = s.Service.Execute(r.Context(), a, rt.Op, r.PathValue("id"), in, r.URL.Query(), id, r.Header.Get("Idempotency-Key"))
			}
		}
	}
	status := 200
	if rt.Op == "order.create" || rt.Op == "seller.product.create" || rt.Op == "address.create" || rt.Op == "seller.customer.create" || rt.Op == "seller.delivery.create" || rt.Op == "extra.request.create" {
		status = 201
	}
	respond(status, data, e)
}
