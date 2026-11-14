package application

import (
	"context"
	"doodhee-backend/internal/domain"
	"doodhee-backend/internal/ports"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

func copyFields(dst, src domain.Record, keys ...string) {
	for _, k := range keys {
		if v, ok := src[k]; ok {
			dst[k] = v
		}
	}
}
func missing() error {
	return domain.Fail(404, "NOT_FOUND", "The record was not found or is not accessible.")
}
func validateName(value string, minimum int, label string) error {
	n := utf8.RuneCountInString(strings.TrimSpace(value))
	// Bound the stored value too, including any surrounding whitespace.
	if n < minimum || utf8.RuneCountInString(value) > 120 {
		return domain.Fail(422, "VALIDATION_ERROR", fmt.Sprintf("%s must contain %d to 120 characters.", label, minimum))
	}
	return nil
}
func validateAddress(r domain.Record) error {
	if n := utf8.RuneCountInString(strings.TrimSpace(r.Text("label"))); n < 1 || n > 40 {
		return domain.Fail(422, "VALIDATION_ERROR", "Address label must contain 1 to 40 characters.")
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(r.Text("address_line"))); n < 5 || n > 300 {
		return domain.Fail(422, "VALIDATION_ERROR", "Address must contain 5 to 300 characters.")
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(r.Text("city"))); n < 2 || n > 100 {
		return domain.Fail(422, "VALIDATION_ERROR", "City must contain 2 to 100 characters.")
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(r.Text("pincode"))); n < 3 || n > 12 {
		return domain.Fail(422, "VALIDATION_ERROR", "Pincode must contain 3 to 12 characters.")
	}
	return nil
}
func (s *Service) own(a domain.Actor, r domain.Record, field string) error {
	if a.Role != "admin" && r.Text(field) != a.ID {
		return missing()
	}
	return nil
}
func (s *Service) Execute(ctx context.Context, a domain.Actor, op, id string, in domain.Record, q url.Values, request, key string) (any, error) {
	role := "customer"
	if strings.HasPrefix(op, "seller.") {
		role = "seller"
	}
	if strings.HasPrefix(op, "admin.") {
		role = "admin"
	}
	if op == "extra.request.get" && a.Role == "seller" {
		role = "seller"
	}
	if op != "me" && op != "profile" && op != "seller.profile" && op != "seller.approval" && op != "seller.profile.update" {
		if e := domain.Require(a, role); e != nil {
			return nil, e
		}
	}
	if strings.HasPrefix(op, "seller.") && a.Role != "seller" {
		return nil, domain.Fail(403, "ROLE_FORBIDDEN", "Seller account required.")
	}
	if op == "me" || op == "seller.profile" || op == "seller.approval" || op == "admin.me" {
		return s.DB.Get(ctx, "users", a.ID, false)
	}
	if op == "profile" || op == "seller.profile.update" {
		if name, ok := in["full_name"]; ok {
			if e := validateName(fmt.Sprint(name), 1, "Full name"); e != nil {
				return nil, e
			}
		}
		if utf8.RuneCountInString(in.Text("pincode")) > 12 {
			return nil, domain.Fail(422, "VALIDATION_ERROR", "Pincode must contain at most 12 characters.")
		}
		r, e := s.DB.Get(ctx, "users", a.ID, false)
		if e != nil {
			return nil, e
		}
		copyFields(r, in, "full_name", "address_line", "city", "pincode")
		r["updated_at"] = now()
		return s.DB.Save(ctx, "users", r)
	}
	switch op {
	case "addresses":
		list, e := s.DB.List(ctx, "addresses", domain.Record{"user_id": a.ID})
		return page(list, q), e
	case "address.create", "address.update", "address.default", "address.delete":
		var out domain.Record
		e := s.DB.Tx(ctx, func(db ports.Store) error {
			if e := db.Lock(ctx, "addresses:"+a.ID); e != nil {
				return e
			}
			list, e := db.List(ctx, "addresses", domain.Record{"user_id": a.ID})
			if e != nil {
				return e
			}
			unsetDefault := func(except string) error {
				for _, address := range list {
					if address.Text("id") == except || !address.Bool("is_default") {
						continue
					}
					address["is_default"] = false
					address["updated_at"] = now()
					if _, saveErr := db.Save(ctx, "addresses", address); saveErr != nil {
						return saveErr
					}
				}
				return nil
			}
			if op == "address.create" {
				if len(list) >= 10 {
					return domain.Fail(409, "ADDRESS_LIMIT", "You can save up to 10 addresses.")
				}
				out = domain.Record{"id": s.Sec.ID(), "user_id": a.ID, "is_default": len(list) == 0}
				copyFields(out, in, "label", "address_line", "city", "pincode", "is_default")
				if len(list) == 0 {
					out["is_default"] = true
				}
				if e = validateAddress(out); e != nil {
					return e
				}
				if out.Bool("is_default") {
					if e = unsetDefault(""); e != nil {
						return e
					}
				}
				out, e = db.Save(ctx, "addresses", out)
				if e != nil {
					return e
				}
				return s.Event(ctx, db, "AddressCreated", "address", out.Text("id"), out)
			}
			out, e = db.Get(ctx, "addresses", id, true)
			if e != nil || out.Text("user_id") != a.ID {
				return missing()
			}
			if op == "address.delete" {
				wasDefault := out.Bool("is_default")
				if e = db.Delete(ctx, "addresses", id); e != nil {
					return e
				}
				if wasDefault {
					remaining, listErr := db.List(ctx, "addresses", domain.Record{"user_id": a.ID})
					if listErr != nil {
						return listErr
					}
					if len(remaining) > 0 {
						remaining[0]["is_default"] = true
						remaining[0]["updated_at"] = now()
						if _, e = db.Save(ctx, "addresses", remaining[0]); e != nil {
							return e
						}
					}
				}
				return s.Event(ctx, db, "AddressDeleted", "address", id, out)
			}
			if op == "address.default" {
				if e = unsetDefault(id); e != nil {
					return e
				}
				out["is_default"] = true
			} else {
				copyFields(out, in, "label", "address_line", "city", "pincode")
				if makeDefault, ok := in["is_default"].(bool); ok && makeDefault {
					if e = unsetDefault(id); e != nil {
						return e
					}
					out["is_default"] = true
				}
				if e = validateAddress(out); e != nil {
					return e
				}
			}
			out["updated_at"] = now()
			out, e = db.Save(ctx, "addresses", out)
			if e != nil {
				return e
			}
			typ := "AddressUpdated"
			if op == "address.default" {
				typ = "DefaultAddressChanged"
			}
			return s.Event(ctx, db, typ, "address", out.Text("id"), out)
		})
		return out, e
	case "sellers":
		list, e := s.rankedSellers(ctx)
		if e != nil {
			return nil, e
		}
		out := []domain.Record{}
		for _, r := range list {
			if q.Get("pincode") != "" && r.Text("pincode") != q.Get("pincode") {
				continue
			}
			out = append(out, r)
		}
		return page(out, q), nil
	case "rankings":
		list, e := s.rankedSellers(ctx)
		return page(list, q), e
	case "discovery.detail":
		list, e := s.rankedSellers(ctx)
		if e != nil {
			return nil, e
		}
		for _, r := range list {
			if r.Text("id") == id {
				return r, nil
			}
		}
		return nil, missing()
	case "rating.get":
		return s.rating(ctx, a, id)
	case "rating.set":
		return s.setRating(ctx, a, id, in)
	case "seller.customers":
		return s.sellerCustomers(ctx, a, q)
	case "seller.customer.create":
		return s.addCustomer(ctx, a, in)
	case "seller.customer.lookup":
		return s.lookupCustomer(ctx, a, q)
	case "seller.customer":
		return s.sellerCustomer(ctx, a, id)
	case "seller.customer.prices":
		return s.customerPrices(ctx, a, id)
	case "seller.customer.price.set":
		return s.setCustomerPrice(ctx, a, id, in, false)
	case "seller.customer.price.delete":
		return s.setCustomerPrice(ctx, a, id, in, true)
	case "products", "product", "seller.products", "seller.product", "admin.products", "admin.product":
		filters := domain.Record{}
		if strings.HasPrefix(op, "seller.") {
			filters["seller_id"] = a.ID
		} else if op == "products" {
			filters["seller_id"] = id
		}
		list, e := s.DB.List(ctx, "products", filters)
		if e != nil {
			return nil, e
		}
		out := []domain.Record{}
		for _, r := range list {
			if !strings.HasPrefix(op, "seller.") && !strings.HasPrefix(op, "admin.") {
				u, e := s.DB.Get(ctx, "users", r.Text("seller_id"), false)
				if e != nil || !u.Bool("is_enabled") || !r.Bool("is_available") || r.Bool("admin_suspended") {
					continue
				}
				r["default_price"] = r["price"]
				price, source, priceErr := s.effectivePrice(ctx, s.DB, r, a.ID)
				if priceErr != nil {
					return nil, priceErr
				}
				r["effective_price"] = price
				r["price_source"] = source
			}
			if strings.HasSuffix(op, ".product") || op == "product" {
				if r.Text("id") == id {
					return r, nil
				}
			} else {
				out = append(out, r)
			}
		}
		if strings.HasSuffix(op, ".product") || op == "product" {
			return nil, missing()
		}
		return page(filter(out, q), q), nil
	case "seller.product.create", "seller.product.update", "seller.product.availability":
		var out domain.Record
		e := s.DB.Tx(ctx, func(db ports.Store) error {
			r := domain.Record{"id": s.Sec.ID(), "seller_id": a.ID, "unit": "litre", "is_available": true}
			if op != "seller.product.create" {
				var e error
				r, e = db.Get(ctx, "products", id, true)
				if e != nil {
					return e
				}
				if e = s.own(a, r, "seller_id"); e != nil {
					return e
				}
			}
			if op == "seller.product.availability" {
				if _, ok := in["is_available"].(bool); !ok {
					return domain.Fail(422, "VALIDATION_ERROR", "Availability must be true or false.")
				}
				copyFields(r, in, "is_available")
			} else {
				copyFields(r, in, "name", "price", "daily_capacity")
				if e := validateName(r.Text("name"), 2, "Product name"); e != nil {
					return e
				}
				if unit, ok := in["unit"]; ok && unit != "litre" {
					return domain.Fail(422, "VALIDATION_ERROR", "Milk products must use the litre unit.")
				}
				if _, e := domain.Decimal(r.Text("price"), 2); e != nil {
					return e
				}
				if _, e := domain.Decimal(r.Text("daily_capacity"), 3); e != nil {
					return e
				}
			}
			r["updated_at"] = now()
			var e error
			out, e = db.Save(ctx, "products", r)
			if e != nil {
				return e
			}
			typ := "ProductCreated"
			if op != "seller.product.create" {
				typ = "ProductAvailabilityChanged"
			}
			return s.Event(ctx, db, typ, "product", out.Text("id"), out)
		})
		return out, e
	case "order.create":
		return s.createOrder(ctx, a, in, key, request)
	case "seller.delivery.create":
		return s.createDelivery(ctx, a, in, key)
	case "orders", "seller.orders", "admin.orders", "order", "seller.order", "admin.order":
		f := domain.Record{}
		if a.Role == "customer" {
			f["customer_id"] = a.ID
		}
		if a.Role == "seller" {
			f["seller_id"] = a.ID
		}
		list, e := s.DB.List(ctx, "orders", f)
		if e != nil {
			return nil, e
		}
		if strings.HasSuffix(op, ".order") || op == "order" {
			for _, r := range list {
				if r.Text("id") == id {
					return r, nil
				}
			}
			return nil, missing()
		}
		return page(filter(list, q), q), nil
	case "order.cancel", "seller.order.accept", "seller.order.dispatch", "seller.order.result", "admin.order.correct":
		return s.changeOrder(ctx, a, op, id, in, key, request)
	case "extra.request.create":
		return s.createExtraRequest(ctx, a, id, in, key)
	case "extra.request.get":
		return s.extraRequest(ctx, a, id)
	case "seller.extra.requests":
		return s.sellerExtraRequests(ctx, a, q)
	case "seller.extra.approve":
		return s.decideExtraRequest(ctx, a, id, true, in, key)
	case "seller.extra.reject":
		return s.decideExtraRequest(ctx, a, id, false, in, key)
	case "summary", "seller.summary", "admin.summary":
		f := domain.Record{}
		if a.Role == "customer" {
			f["customer_id"] = a.ID
		}
		if a.Role == "seller" {
			f["seller_id"] = a.ID
		}
		list, e := s.DB.List(ctx, "orders", f)
		if e != nil {
			return nil, e
		}
		total, litres := new(big.Rat), new(big.Rat)
		days := map[string]bool{}
		groups := map[string]domain.Record{}
		for _, r := range filter(list, q) {
			if r.Text("status") != "delivered" {
				continue
			}
			v, _ := new(big.Rat).SetString(r.Text("total_amount"))
			l, _ := new(big.Rat).SetString(r.Text("quantity"))
			total.Add(total, v)
			litres.Add(litres, l)
			days[r.Text("service_date")] = true
			k := r.Text("customer_id")
			if a.Role == "customer" {
				k = r.Text("seller_id")
			}
			g := groups[k]
			if g == nil {
				g = domain.Record{"user_id": k, "total_amount": "0.00", "delivered_litres": "0.000"}
				if person, e := s.DB.Get(ctx, "users", k, false); e == nil {
					g["full_name"] = person.Text("full_name")
				}
			}
			x, _ := new(big.Rat).SetString(g.Text("total_amount"))
			y, _ := new(big.Rat).SetString(g.Text("delivered_litres"))
			g["total_amount"] = x.Add(x, v).FloatString(2)
			g["delivered_litres"] = y.Add(y, l).FloatString(3)
			groups[k] = g
		}
		return domain.Record{"delivered_days": len(days), "delivered_litres": litres.FloatString(3), "total_amount": total.FloatString(2), "groups": groups}, nil
	case "admin.users", "admin.user":
		f := domain.Record{}
		if op == "admin.user" {
			return s.DB.Get(ctx, "users", id, false)
		}
		rows, e := s.DB.List(ctx, "users", f)
		return page(filter(rows, q), q), e
	case "admin.enable", "admin.disable", "admin.product.suspend", "admin.product.activate", "admin.event.retry":
		return s.adminChange(ctx, a, op, id, in, request)
	case "admin.events", "admin.event", "admin.audit":
		table := "outbox_events"
		if op == "admin.audit" {
			table = "admin_audit_logs"
		}
		if op == "admin.event" {
			return s.DB.Get(ctx, table, id, false)
		}
		rows, e := s.DB.List(ctx, table, domain.Record{})
		return page(filter(rows, q), q), e
	case "admin.dashboard":
		out := domain.Record{}
		for _, t := range []string{"users", "products", "orders", "outbox_events"} {
			rows, e := s.DB.List(ctx, t, domain.Record{})
			if e != nil {
				return nil, e
			}
			out[t] = len(rows)
		}
		return out, nil
	}
	return nil, domain.Fail(404, "NOT_FOUND", "Operation not found.")
}
func filter(rows []domain.Record, q url.Values) []domain.Record {
	out := []domain.Record{}
	for _, r := range rows {
		ok := true
		for _, k := range []string{"status", "seller_id", "customer_id", "role", "approval_status", "is_enabled", "is_available", "phone", "event_type", "admin_id", "action", "target_id"} {
			if q.Get(k) != "" && r.Text(k) != q.Get(k) {
				ok = false
			}
		}
		date := r.Text("service_date")
		if date == "" {
			date = r.Text("created_at")
			if len(date) > 10 {
				date = date[:10]
			}
		}
		if q.Get("service_date") != "" && date != q.Get("service_date") {
			ok = false
		}
		if q.Get("from") != "" && date < q.Get("from") {
			ok = false
		}
		if q.Get("to") != "" && date > q.Get("to") {
			ok = false
		}
		if ok {
			out = append(out, r)
		}
	}
	return out
}
func page(rows []domain.Record, q url.Values) domain.Record {
	start := 0
	if c := q.Get("cursor"); c != "" {
		for i, r := range rows {
			if r.Text("id") == c {
				start = i + 1
			}
		}
	}
	end := start + 50
	if end > len(rows) {
		end = len(rows)
	}
	next := ""
	if end < len(rows) {
		next = rows[end-1].Text("id")
	}
	return domain.Record{"items": rows[start:end], "next_cursor": next}
}
func (s *Service) idempotent(ctx context.Context, a domain.Actor, route, key string, in domain.Record, fn func(ports.Store) (domain.Record, error)) (domain.Record, error) {
	if len(key) < 8 || len(key) > 160 {
		return nil, domain.Fail(422, "IDEMPOTENCY_REQUIRED", "A valid Idempotency-Key is required.")
	}
	var out domain.Record
	b, _ := json.Marshal(in)
	hash := s.Sec.Hash(string(b))
	e := s.DB.Tx(ctx, func(db ports.Store) error {
		if e := db.Lock(ctx, a.ID+route+key); e != nil {
			return e
		}
		existing, e := db.List(ctx, "idempotency_keys", domain.Record{"user_id": a.ID, "key": key, "route": route})
		if e != nil {
			return e
		}
		if len(existing) > 0 {
			r := existing[0]
			if r.Text("request_hash") != hash {
				return domain.Fail(409, "IDEMPOTENCY_CONFLICT", "This request key was used with different input.")
			}
			raw, _ := json.Marshal(r["response"])
			return json.Unmarshal(raw, &out)
		}
		out, e = fn(db)
		if e != nil {
			return e
		}
		raw, _ := json.Marshal(out)
		_, e = db.Save(ctx, "idempotency_keys", domain.Record{"id": s.Sec.ID(), "user_id": a.ID, "key": key, "route": route, "request_hash": hash, "response": string(raw)})
		return e
	})
	return out, e
}
func (s *Service) createOrder(ctx context.Context, a domain.Actor, in domain.Record, key, req string) (domain.Record, error) {
	return s.idempotent(ctx, a, "createOrder", key, in, func(db ports.Store) (domain.Record, error) {
		p, e := db.Get(ctx, "products", in.Text("product_id"), true)
		if e != nil {
			return nil, e
		}
		u, e := db.Get(ctx, "users", p.Text("seller_id"), false)
		if e != nil {
			return nil, e
		}
		if !p.Bool("is_available") || p.Bool("admin_suspended") || !u.Bool("is_enabled") {
			return nil, domain.Fail(409, "PRODUCT_UNAVAILABLE", "This product is unavailable.")
		}
		date, e := time.Parse("2006-01-02", in.Text("service_date"))
		if e != nil || date.Format("2006-01-02") < time.Now().In(time.FixedZone("India", 19800)).Format("2006-01-02") || date.After(time.Now().AddDate(0, 0, 90)) {
			return nil, domain.Fail(422, "VALIDATION_ERROR", "Select a delivery date within 90 days.")
		}
		if len(in.Text("delivery_address")) < 5 {
			return nil, domain.Fail(422, "VALIDATION_ERROR", "A delivery address is required.")
		}
		price, priceSource, e := s.effectivePrice(ctx, db, p, a.ID)
		if e != nil {
			return nil, e
		}
		total, e := domain.Total(in.Text("quantity"), price)
		if e != nil {
			return nil, e
		}
		if e = s.ensureCapacity(ctx, db, p.Text("id"), in.Text("service_date"), in.Text("quantity"), ""); e != nil {
			return nil, e
		}
		r := domain.Record{"id": s.Sec.ID(), "customer_id": a.ID, "seller_id": p.Text("seller_id"), "product_id": p.Text("id"), "service_date": in.Text("service_date"), "quantity": in.Text("quantity"), "base_quantity": in.Text("quantity"), "approved_extra_quantity": "0.000", "unit_price": price, "price_source": priceSource, "total_amount": total, "status": "pending", "delivery_address": in.Text("delivery_address"), "customer_note": in.Text("customer_note")}
		out, e := db.Save(ctx, "orders", r)
		if e == nil {
			e = s.Event(ctx, db, "OrderPlaced", "order", out.Text("id"), out)
		}
		return out, e
	})
}
func (s *Service) changeOrder(ctx context.Context, a domain.Actor, op, id string, in domain.Record, key, req string) (domain.Record, error) {
	fn := func(db ports.Store) (domain.Record, error) {
		r, e := db.Get(ctx, "orders", id, true)
		if e != nil {
			return nil, e
		}
		field := "seller_id"
		if a.Role == "customer" {
			field = "customer_id"
		}
		if e = s.own(a, r, field); e != nil {
			return nil, e
		}
		before := domain.Record{}
		copyFields(before, r, "id", "status", "not_delivered_reason", "delivery_note")
		next := in.Text("result")
		switch op {
		case "order.cancel":
			next = "cancelled"
		case "seller.order.accept":
			next = "accepted"
		case "seller.order.dispatch":
			next = "out_for_delivery"
		}
		admin := op == "admin.order.correct"
		if op == "order.cancel" && strings.TrimSpace(in.Text("reason")) == "" {
			return nil, domain.Fail(422, "REASON_REQUIRED", "A cancellation reason is required.")
		}
		if admin && len(strings.TrimSpace(in.Text("reason"))) < 3 {
			return nil, domain.Fail(422, "REASON_REQUIRED", "A correction reason is required.")
		}
		if e = domain.Transition(r.Text("status"), next, in.Text("reason_code"), admin); e != nil {
			return nil, e
		}
		if next == "delivered" {
			quantity := r.Text("quantity")
			if in.Text("quantity") != "" {
				quantity = in.Text("quantity")
			}
			if _, e = domain.Decimal(quantity, 3); e != nil {
				return nil, e
			}
			if e = s.ensureCapacity(ctx, db, r.Text("product_id"), r.Text("service_date"), quantity, r.Text("id")); e != nil {
				return nil, e
			}
			price := r.Text("unit_price")
			if in.Text("unit_price") != "" {
				if _, e = domain.Decimal(in.Text("unit_price"), 2); e != nil {
					return nil, e
				}
				price = in.Text("unit_price")
				r["price_source"] = "delivery"
			}
			r["quantity"] = quantity
			r["unit_price"] = price
			r["total_amount"], e = domain.Total(quantity, price)
			if e != nil {
				return nil, e
			}
		}
		r["status"] = next
		r["updated_at"] = now()
		r["delivery_note"] = in.Text("note")
		if next == "cancelled" {
			r["delivery_note"] = in.Text("reason")
		}
		r["not_delivered_reason"] = nil
		if next == "not_delivered" {
			r["not_delivered_reason"] = in.Text("reason_code")
		}
		if next == "delivered" || next == "not_delivered" {
			r["marked_by"] = a.ID
			r["delivery_marked_at"] = now()
		}
		out, e := db.Save(ctx, "orders", r)
		if e != nil {
			return nil, e
		}
		if admin {
			if e = s.Audit(ctx, db, a, op, "orders", id, in.Text("reason"), req, before, out); e != nil {
				return nil, e
			}
		}
		typ := map[string]string{"accepted": "OrderAccepted", "out_for_delivery": "OrderOutForDelivery", "delivered": "OrderDelivered", "not_delivered": "OrderNotDelivered", "cancelled": "OrderCancelled"}[next]
		e = s.Event(ctx, db, typ, "order", id, out)
		return out, e
	}
	if op == "admin.order.correct" || op == "seller.order.result" {
		return s.idempotent(ctx, a, op+id, key, in, fn)
	}
	var out domain.Record
	e := s.DB.Tx(ctx, func(db ports.Store) error { var e error; out, e = fn(db); return e })
	return out, e
}
func (s *Service) adminChange(ctx context.Context, a domain.Actor, op, id string, in domain.Record, req string) (domain.Record, error) {
	var out domain.Record
	e := s.DB.Tx(ctx, func(db ports.Store) error {
		table := "users"
		if strings.Contains(op, "product.") {
			table = "products"
		}
		if strings.Contains(op, "event.") {
			table = "outbox_events"
		}
		r, e := db.Get(ctx, table, id, true)
		if e != nil {
			return e
		}
		b, _ := json.Marshal(r)
		var before domain.Record
		json.Unmarshal(b, &before)
		reason := in.Text("reason")
		if len(strings.TrimSpace(reason)) < 3 {
			return domain.Fail(422, "REASON_REQUIRED", "A reason of at least three characters is required.")
		}
		typ := "UserUpdated"
		switch op {
		case "admin.enable", "admin.disable":
			if r.Text("role") == "admin" {
				return domain.Fail(403, "ROLE_FORBIDDEN", "Admin account changes require the trusted CLI.")
			}
			r["is_enabled"] = op == "admin.enable"
			if op == "admin.disable" {
				typ = "UserDisabled"
				sessions, e := db.List(ctx, "auth_sessions", domain.Record{"user_id": id})
				if e != nil {
					return e
				}
				for _, sess := range sessions {
					sess["revoked_at"] = now()
					if _, e = db.Save(ctx, "auth_sessions", sess); e != nil {
						return e
					}
				}
			}
		case "admin.product.suspend", "admin.product.activate":
			r["admin_suspended"] = op == "admin.product.suspend"
			typ = "ProductAvailabilityChanged"
		case "admin.event.retry":
			if r.Text("status") != "failed" {
				return domain.Fail(409, "EVENT_NOT_FAILED", "Only failed events can be retried.")
			}
			r["status"] = "pending"
			r["attempt_count"] = 0
			r["next_attempt_at"] = now()
			typ = "EventRetried"
		}
		if table != "outbox_events" {
			r["updated_at"] = now()
		}
		out, e = db.Save(ctx, table, r)
		if e != nil {
			return e
		}
		if e = s.Audit(ctx, db, a, op, table, id, reason, req, before, out); e != nil {
			return e
		}
		return s.Event(ctx, db, typ, table, id, out)
	})
	return out, e
}

func (s *Service) SeedAdmin(ctx context.Context, phone, name, secret string) error {
	return s.DB.Tx(ctx, func(db ports.Store) error {
		users, e := db.List(ctx, "users", domain.Record{"phone": phone})
		if e != nil {
			return e
		}
		if len(users) > 0 {
			return fmt.Errorf("account already exists; admin seed is create-only")
		}
		u, e := db.Save(ctx, "users", domain.Record{"id": s.Sec.ID(), "phone": phone, "full_name": name, "role": "admin", "approval_status": "approved", "phone_verified_at": now()})
		if e != nil {
			return e
		}
		encrypted, e := s.Sec.Encrypt(secret)
		if e != nil {
			return e
		}
		_, e = db.Save(ctx, "mfa_methods", domain.Record{"id": s.Sec.ID(), "user_id": u.Text("id"), "secret_encrypted": encrypted})
		return e
	})
}
