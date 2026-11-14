package application

import (
	"context"
	"doodhee-backend/internal/domain"
	"doodhee-backend/internal/ports"
	"fmt"
	"math/big"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

func indiaToday() string {
	return time.Now().In(time.FixedZone("India", 19800)).Format("2006-01-02")
}

func (s *Service) relationship(ctx context.Context, db ports.Store, milkmanID, customerID string, lock bool) (domain.Record, error) {
	rows, err := db.List(ctx, "milkman_customers", domain.Record{"milkman_id": milkmanID, "customer_id": customerID})
	if err != nil || len(rows) == 0 {
		return nil, missing()
	}
	r := rows[0]
	if lock {
		r, err = db.Get(ctx, "milkman_customers", r.Text("id"), true)
	}
	if err != nil || !r.Bool("is_active") {
		return nil, missing()
	}
	return r, nil
}

func (s *Service) effectivePrice(ctx context.Context, db ports.Store, product domain.Record, customerID string) (string, string, error) {
	rows, err := db.List(ctx, "customer_product_prices", domain.Record{
		"milkman_id": product.Text("seller_id"), "customer_id": customerID, "product_id": product.Text("id"),
	})
	if err != nil {
		return "", "", err
	}
	if len(rows) > 0 {
		return rows[0].Text("price"), "customer", nil
	}
	return product.Text("price"), "default", nil
}

func (s *Service) ensureCapacity(ctx context.Context, db ports.Store, productID, serviceDate, quantity, excludeOrderID string) error {
	orders, err := db.List(ctx, "orders", domain.Record{"product_id": productID, "service_date": serviceDate})
	if err != nil {
		return err
	}
	product, err := db.Get(ctx, "products", productID, true)
	if err != nil {
		return err
	}
	used := new(big.Rat)
	for _, order := range orders {
		if order.Text("id") == excludeOrderID || order.Text("status") == "cancelled" || order.Text("status") == "not_delivered" {
			continue
		}
		value, _ := new(big.Rat).SetString(order.Text("quantity"))
		used.Add(used, value)
	}
	wanted, _ := new(big.Rat).SetString(quantity)
	capacity, _ := new(big.Rat).SetString(product.Text("daily_capacity"))
	if used.Add(used, wanted).Cmp(capacity) > 0 {
		return domain.Fail(409, "CAPACITY_EXCEEDED", "The Milkman has no remaining capacity for this day.")
	}
	return nil
}

func (s *Service) customerView(ctx context.Context, db ports.Store, relation domain.Record) (domain.Record, error) {
	user, err := db.Get(ctx, "users", relation.Text("customer_id"), false)
	if err != nil {
		return nil, err
	}
	return domain.Record{
		"id": user["id"], "full_name": user["full_name"], "phone": user["phone"],
		"phone_verified_at": user["phone_verified_at"], "relationship_id": relation["id"],
		"delivery_address": relation["delivery_address"], "is_active": relation["is_active"],
		"created_at": relation["created_at"], "updated_at": relation["updated_at"],
	}, nil
}

func (s *Service) sellerCustomers(ctx context.Context, a domain.Actor, q url.Values) (domain.Record, error) {
	relations, err := s.DB.List(ctx, "milkman_customers", domain.Record{"milkman_id": a.ID, "is_active": "true"})
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(strings.TrimSpace(q.Get("search")))
	rows := []domain.Record{}
	for _, relation := range relations {
		view, viewErr := s.customerView(ctx, s.DB, relation)
		if viewErr != nil {
			return nil, viewErr
		}
		haystack := strings.ToLower(view.Text("full_name") + " " + view.Text("phone") + " " + view.Text("delivery_address"))
		if needle == "" || strings.Contains(haystack, needle) {
			rows = append(rows, view)
		}
	}
	return page(rows, q), nil
}

func (s *Service) lookupCustomer(ctx context.Context, a domain.Actor, q url.Values) (domain.Record, error) {
	phone := strings.TrimSpace(q.Get("phone"))
	if !validPhone(phone) {
		return nil, domain.Fail(422, "VALIDATION_ERROR", "Enter a valid phone number including country code.")
	}
	users, err := s.DB.List(ctx, "users", domain.Record{"phone": phone})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return domain.Record{"exists": false, "phone": phone, "already_connected": false}, nil
	}
	customer := users[0]
	if customer.Text("role") != "customer" {
		return nil, domain.Fail(409, "PHONE_ALREADY_IN_USE", "This mobile number belongs to a non-customer account.")
	}
	out := domain.Record{
		"exists":            true,
		"id":                customer["id"],
		"full_name":         customer["full_name"],
		"phone":             customer["phone"],
		"phone_verified_at": customer["phone_verified_at"],
		"already_connected": false,
	}
	relations, err := s.DB.List(ctx, "milkman_customers", domain.Record{"milkman_id": a.ID, "customer_id": customer.Text("id")})
	if err != nil {
		return nil, err
	}
	if len(relations) > 0 && relations[0].Bool("is_active") {
		out["already_connected"] = true
		out["delivery_address"] = relations[0]["delivery_address"]
	}
	return out, nil
}

func (s *Service) addCustomer(ctx context.Context, a domain.Actor, in domain.Record) (domain.Record, error) {
	if err := validateName(in.Text("full_name"), 1, "Customer name"); err != nil {
		return nil, err
	}
	if !validPhone(in.Text("phone")) {
		return nil, domain.Fail(422, "VALIDATION_ERROR", "Enter a valid phone number including country code.")
	}
	if n := len(strings.TrimSpace(in.Text("delivery_address"))); n < 5 || n > 300 {
		return nil, domain.Fail(422, "VALIDATION_ERROR", "Delivery address must contain 5 to 300 characters.")
	}
	var out domain.Record
	err := s.DB.Tx(ctx, func(db ports.Store) error {
		if err := db.Lock(ctx, "user:"+in.Text("phone")); err != nil {
			return err
		}
		users, err := db.List(ctx, "users", domain.Record{"phone": in.Text("phone")})
		if err != nil {
			return err
		}
		var customer domain.Record
		created := false
		if len(users) == 0 {
			customer, err = db.Save(ctx, "users", domain.Record{
				"id": s.Sec.ID(), "phone": in.Text("phone"), "full_name": strings.TrimSpace(in.Text("full_name")),
				"role": "customer", "approval_status": "approved", "address_line": strings.TrimSpace(in.Text("delivery_address")),
			})
			created = true
		} else {
			customer = users[0]
			if customer.Text("role") != "customer" {
				return domain.Fail(409, "PHONE_ALREADY_IN_USE", "This mobile number belongs to a non-customer account.")
			}
		}
		if err != nil {
			return err
		}
		relations, err := db.List(ctx, "milkman_customers", domain.Record{"milkman_id": a.ID, "customer_id": customer.Text("id")})
		if err != nil {
			return err
		}
		var relation domain.Record
		if len(relations) > 0 {
			relation = relations[0]
			relation["delivery_address"] = strings.TrimSpace(in.Text("delivery_address"))
			relation["is_active"] = true
			relation["updated_at"] = now()
		} else {
			relation = domain.Record{"id": s.Sec.ID(), "milkman_id": a.ID, "customer_id": customer.Text("id"), "delivery_address": strings.TrimSpace(in.Text("delivery_address")), "is_active": true}
		}
		relation, err = db.Save(ctx, "milkman_customers", relation)
		if err != nil {
			return err
		}
		out, err = s.customerView(ctx, db, relation)
		if err != nil {
			return err
		}
		if created {
			if err = s.Event(ctx, db, "CustomerPreRegistered", "user", customer.Text("id"), customer); err != nil {
				return err
			}
		}
		return s.Event(ctx, db, "MilkmanCustomerConnected", "milkman_customer", relation.Text("id"), relation)
	})
	return out, err
}

func (s *Service) sellerCustomer(ctx context.Context, a domain.Actor, customerID string) (domain.Record, error) {
	relation, err := s.relationship(ctx, s.DB, a.ID, customerID, false)
	if err != nil {
		return nil, err
	}
	return s.customerView(ctx, s.DB, relation)
}

func (s *Service) customerPrices(ctx context.Context, a domain.Actor, customerID string) (domain.Record, error) {
	if _, err := s.relationship(ctx, s.DB, a.ID, customerID, false); err != nil {
		return nil, err
	}
	rows, err := s.DB.List(ctx, "customer_product_prices", domain.Record{"milkman_id": a.ID, "customer_id": customerID})
	return domain.Record{"items": rows, "next_cursor": ""}, err
}

func (s *Service) setCustomerPrice(ctx context.Context, a domain.Actor, customerID string, in domain.Record, remove bool) (domain.Record, error) {
	var out domain.Record
	err := s.DB.Tx(ctx, func(db ports.Store) error {
		if _, err := s.relationship(ctx, db, a.ID, customerID, true); err != nil {
			return err
		}
		product, err := db.Get(ctx, "products", in.Text("product_id"), false)
		if err != nil || product.Text("seller_id") != a.ID {
			return missing()
		}
		rows, err := db.List(ctx, "customer_product_prices", domain.Record{"milkman_id": a.ID, "customer_id": customerID, "product_id": product.Text("id")})
		if err != nil {
			return err
		}
		if remove {
			if len(rows) == 0 {
				return missing()
			}
			out = rows[0]
			if err = db.Delete(ctx, "customer_product_prices", out.Text("id")); err != nil {
				return err
			}
			return s.Event(ctx, db, "CustomerPriceRemoved", "customer_product_price", out.Text("id"), out)
		}
		if _, err = domain.Decimal(in.Text("price"), 2); err != nil {
			return err
		}
		if len(rows) > 0 {
			out = rows[0]
			out["price"] = in.Text("price")
			out["updated_at"] = now()
		} else {
			out = domain.Record{"id": s.Sec.ID(), "milkman_id": a.ID, "customer_id": customerID, "product_id": product.Text("id"), "price": in.Text("price")}
		}
		out, err = db.Save(ctx, "customer_product_prices", out)
		if err != nil {
			return err
		}
		return s.Event(ctx, db, "CustomerPriceChanged", "customer_product_price", out.Text("id"), out)
	})
	return out, err
}

func (s *Service) createDelivery(ctx context.Context, a domain.Actor, in domain.Record, key string) (domain.Record, error) {
	return s.idempotent(ctx, a, "createDelivery", key, in, func(db ports.Store) (domain.Record, error) {
		relation, err := s.relationship(ctx, db, a.ID, in.Text("customer_id"), true)
		if err != nil {
			return nil, err
		}
		product, err := db.Get(ctx, "products", in.Text("product_id"), true)
		if err != nil || product.Text("seller_id") != a.ID {
			return nil, missing()
		}
		if !product.Bool("is_available") || product.Bool("admin_suspended") {
			return nil, domain.Fail(409, "PRODUCT_UNAVAILABLE", "This product is unavailable.")
		}
		if _, err = domain.Decimal(in.Text("quantity"), 3); err != nil {
			return nil, err
		}
		date, err := time.Parse("2006-01-02", in.Text("service_date"))
		today, _ := time.Parse("2006-01-02", indiaToday())
		if err != nil || date.After(today) || date.Before(today.AddDate(0, 0, -90)) {
			return nil, domain.Fail(422, "VALIDATION_ERROR", "Select today or a date within the last 90 days.")
		}
		if err = s.ensureCapacity(ctx, db, product.Text("id"), in.Text("service_date"), in.Text("quantity"), ""); err != nil {
			return nil, err
		}
		price, source, err := s.effectivePrice(ctx, db, product, in.Text("customer_id"))
		if err != nil {
			return nil, err
		}
		if in.Text("unit_price") != "" {
			if _, err = domain.Decimal(in.Text("unit_price"), 2); err != nil {
				return nil, err
			}
			price, source = in.Text("unit_price"), "delivery"
		}
		total, err := domain.Total(in.Text("quantity"), price)
		if err != nil {
			return nil, err
		}
		address := strings.TrimSpace(in.Text("delivery_address"))
		if address == "" {
			address = relation.Text("delivery_address")
		}
		order := domain.Record{
			"id": s.Sec.ID(), "customer_id": in.Text("customer_id"), "seller_id": a.ID, "product_id": product.Text("id"),
			"service_date": in.Text("service_date"), "quantity": in.Text("quantity"), "base_quantity": in.Text("quantity"),
			"approved_extra_quantity": "0.000", "unit_price": price, "price_source": source, "total_amount": total,
			"status": "delivered", "delivery_address": address, "delivery_note": in.Text("note"), "customer_note": "",
			"marked_by": a.ID, "delivery_marked_at": now(),
		}
		out, err := db.Save(ctx, "orders", order)
		if err == nil {
			err = s.Event(ctx, db, "DeliveryRecorded", "order", out.Text("id"), out)
		}
		return out, err
	})
}

func (s *Service) rating(ctx context.Context, a domain.Actor, milkmanID string) (domain.Record, error) {
	rows, err := s.DB.List(ctx, "milkman_ratings", domain.Record{"customer_id": a.ID, "milkman_id": milkmanID})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return domain.Record{"customer_id": a.ID, "milkman_id": milkmanID, "rating": nil}, nil
	}
	return rows[0], nil
}

func (s *Service) setRating(ctx context.Context, a domain.Actor, milkmanID string, in domain.Record) (domain.Record, error) {
	value, err := strconv.Atoi(in.Text("rating"))
	if err != nil || value < 1 || value > 5 {
		return nil, domain.Fail(422, "VALIDATION_ERROR", "Rating must be a whole number from 1 to 5.")
	}
	var out domain.Record
	err = s.DB.Tx(ctx, func(db ports.Store) error {
		milkman, getErr := db.Get(ctx, "users", milkmanID, false)
		if getErr != nil || milkman.Text("role") != "seller" || !milkman.Bool("is_enabled") {
			return missing()
		}
		if lockErr := db.Lock(ctx, "rating:"+a.ID+":"+milkmanID); lockErr != nil {
			return lockErr
		}
		rows, listErr := db.List(ctx, "milkman_ratings", domain.Record{"customer_id": a.ID, "milkman_id": milkmanID})
		if listErr != nil {
			return listErr
		}
		if len(rows) > 0 {
			out = rows[0]
			out["rating"] = value
			out["updated_at"] = now()
		} else {
			out = domain.Record{"id": s.Sec.ID(), "customer_id": a.ID, "milkman_id": milkmanID, "rating": value}
		}
		out, listErr = db.Save(ctx, "milkman_ratings", out)
		if listErr != nil {
			return listErr
		}
		return s.Event(ctx, db, "MilkmanRated", "milkman_rating", out.Text("id"), out)
	})
	return out, err
}

func (s *Service) rankedSellers(ctx context.Context) ([]domain.Record, error) {
	users, err := s.DB.List(ctx, "users", domain.Record{"role": "seller", "is_enabled": "true"})
	if err != nil {
		return nil, err
	}
	ratings, err := s.DB.List(ctx, "milkman_ratings", domain.Record{})
	if err != nil {
		return nil, err
	}
	type score struct{ count, sum int }
	scores := map[string]score{}
	for _, rating := range ratings {
		value, _ := strconv.Atoi(rating.Text("rating"))
		current := scores[rating.Text("milkman_id")]
		current.count++
		current.sum += value
		scores[rating.Text("milkman_id")] = current
	}
	sort.Slice(users, func(i, j int) bool {
		a, b := scores[users[i].Text("id")], scores[users[j].Text("id")]
		left, right := a.sum*b.count, b.sum*a.count
		if a.count == 0 || b.count == 0 {
			left, right = a.sum, b.sum
		}
		if left != right {
			return left > right
		}
		if a.count != b.count {
			return a.count > b.count
		}
		return strings.ToLower(users[i].Text("full_name")) < strings.ToLower(users[j].Text("full_name"))
	})
	out := make([]domain.Record, 0, len(users))
	for index, user := range users {
		score := scores[user.Text("id")]
		average := "0.00"
		if score.count > 0 {
			average = fmt.Sprintf("%.2f", float64(score.sum)/float64(score.count))
		}
		out = append(out, domain.Record{"id": user["id"], "full_name": user["full_name"], "city": user["city"], "pincode": user["pincode"], "average_rating": average, "rating_count": score.count, "rank": index + 1})
	}
	return out, nil
}

func (s *Service) createExtraRequest(ctx context.Context, a domain.Actor, orderID string, in domain.Record, key string) (domain.Record, error) {
	return s.idempotent(ctx, a, "createExtraRequest:"+orderID, key, in, func(db ports.Store) (domain.Record, error) {
		order, err := db.Get(ctx, "orders", orderID, true)
		if err != nil || order.Text("customer_id") != a.ID {
			return nil, missing()
		}
		if order.Text("status") != "pending" && order.Text("status") != "accepted" {
			return nil, domain.Fail(409, "ORDER_NOT_UPCOMING", "Extra milk can only be requested for an upcoming delivery.")
		}
		if order.Text("service_date") < indiaToday() {
			return nil, domain.Fail(409, "ORDER_NOT_UPCOMING", "This delivery date has passed.")
		}
		if _, err = domain.Decimal(in.Text("extra_quantity"), 3); err != nil {
			return nil, err
		}
		existing, err := db.List(ctx, "extra_milk_requests", domain.Record{"order_id": orderID})
		if err != nil {
			return nil, err
		}
		for _, request := range existing {
			if request.Text("status") == "pending" || request.Text("status") == "approved" {
				return nil, domain.Fail(409, "EXTRA_REQUEST_EXISTS", "An extra milk request already exists for this delivery.")
			}
		}
		request := domain.Record{"id": s.Sec.ID(), "order_id": orderID, "customer_id": a.ID, "milkman_id": order.Text("seller_id"), "extra_quantity": in.Text("extra_quantity"), "status": "pending"}
		out, err := db.Save(ctx, "extra_milk_requests", request)
		if err == nil {
			err = s.Event(ctx, db, "ExtraMilkRequested", "extra_milk_request", out.Text("id"), out)
		}
		return out, err
	})
}

func (s *Service) extraRequest(ctx context.Context, a domain.Actor, orderID string) (domain.Record, error) {
	order, err := s.DB.Get(ctx, "orders", orderID, false)
	if err != nil || (order.Text("customer_id") != a.ID && order.Text("seller_id") != a.ID) {
		return nil, missing()
	}
	rows, err := s.DB.List(ctx, "extra_milk_requests", domain.Record{"order_id": orderID})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return domain.Record{"order_id": orderID, "status": "none"}, nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Text("created_at") > rows[j].Text("created_at") })
	return rows[0], nil
}

func (s *Service) sellerExtraRequests(ctx context.Context, a domain.Actor, q url.Values) (domain.Record, error) {
	rows, err := s.DB.List(ctx, "extra_milk_requests", domain.Record{"milkman_id": a.ID})
	if err != nil {
		return nil, err
	}
	for _, request := range rows {
		order, orderErr := s.DB.Get(ctx, "orders", request.Text("order_id"), false)
		if orderErr != nil || order.Text("seller_id") != a.ID {
			return nil, missing()
		}
		customer, customerErr := s.DB.Get(ctx, "users", request.Text("customer_id"), false)
		if customerErr != nil {
			return nil, customerErr
		}
		product, productErr := s.DB.Get(ctx, "products", order.Text("product_id"), false)
		if productErr != nil {
			return nil, productErr
		}
		request["service_date"] = order["service_date"]
		request["planned_quantity"] = order["quantity"]
		request["base_quantity"] = order["base_quantity"]
		request["full_name"] = customer["full_name"]
		request["phone"] = customer["phone"]
		request["product_name"] = product["name"]
	}
	return page(filter(rows, q), q), nil
}

func (s *Service) decideExtraRequest(ctx context.Context, a domain.Actor, requestID string, approve bool, in domain.Record, key string) (domain.Record, error) {
	action := "reject"
	if approve {
		action = "approve"
	}
	return s.idempotent(ctx, a, "extraRequest:"+action+":"+requestID, key, in, func(db ports.Store) (domain.Record, error) {
		request, err := db.Get(ctx, "extra_milk_requests", requestID, true)
		if err != nil || request.Text("milkman_id") != a.ID {
			return nil, missing()
		}
		if request.Text("status") != "pending" {
			return nil, domain.Fail(409, "EXTRA_REQUEST_DECIDED", "This extra milk request has already been decided.")
		}
		order, err := db.Get(ctx, "orders", request.Text("order_id"), true)
		if err != nil || order.Text("seller_id") != a.ID || (order.Text("status") != "pending" && order.Text("status") != "accepted") {
			return nil, domain.Fail(409, "ORDER_NOT_UPCOMING", "The related delivery is no longer open.")
		}
		request["status"] = "rejected"
		request["decision_note"] = in.Text("decision_note")
		request["decided_at"] = now()
		request["updated_at"] = now()
		if approve {
			planned, _ := new(big.Rat).SetString(order.Text("quantity"))
			extra, _ := new(big.Rat).SetString(request.Text("extra_quantity"))
			quantity := planned.Add(planned, extra).FloatString(3)
			if err = s.ensureCapacity(ctx, db, order.Text("product_id"), order.Text("service_date"), quantity, order.Text("id")); err != nil {
				return nil, err
			}
			approved, _ := new(big.Rat).SetString(order.Text("approved_extra_quantity"))
			order["quantity"] = quantity
			order["approved_extra_quantity"] = approved.Add(approved, extra).FloatString(3)
			order["total_amount"], err = domain.Total(quantity, order.Text("unit_price"))
			if err != nil {
				return nil, err
			}
			order["updated_at"] = now()
			if _, err = db.Save(ctx, "orders", order); err != nil {
				return nil, err
			}
			request["status"] = "approved"
		}
		out, err := db.Save(ctx, "extra_milk_requests", request)
		if err != nil {
			return nil, err
		}
		typ := "ExtraMilkRejected"
		if approve {
			typ = "ExtraMilkApproved"
		}
		return out, s.Event(ctx, db, typ, "extra_milk_request", out.Text("id"), out)
	})
}
