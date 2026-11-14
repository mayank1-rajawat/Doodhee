package api

// Input fields are shared by HTTP validation and OpenAPI generation.
var Fields = map[string][]string{
	"auth.request": {"phone", "role", "mode"}, "auth.verify": {"challenge_id", "code", "full_name"}, "auth.mfa": {"challenge_id", "code"}, "auth.refresh": {"refresh_token"},
	"profile": {"full_name", "address_line", "city", "pincode"}, "seller.profile.update": {"full_name", "address_line", "city", "pincode"},
	"address.create": {"label", "address_line", "city", "pincode", "is_default"}, "address.update": {"label", "address_line", "city", "pincode", "is_default"}, "address.default": {}, "address.delete": {},
	"seller.product.create": {"name", "unit", "price", "daily_capacity"}, "seller.product.update": {"name", "price", "daily_capacity"}, "seller.product.availability": {"is_available"},
	"order.create": {"product_id", "service_date", "quantity", "delivery_address", "customer_note"}, "order.cancel": {"reason"},
	"seller.customer.create": {"full_name", "phone", "delivery_address"}, "seller.customer.price.set": {"product_id", "price"}, "seller.customer.price.delete": {"product_id"},
	"seller.delivery.create": {"customer_id", "product_id", "service_date", "quantity", "unit_price", "delivery_address", "note"},
	"seller.order.result":    {"result", "reason_code", "note", "quantity", "unit_price"}, "admin.order.correct": {"result", "reason_code", "note", "reason"},
	"rating.set": {"rating"}, "extra.request.create": {"extra_quantity"}, "seller.extra.approve": {"decision_note"}, "seller.extra.reject": {"decision_note"},
	"admin.enable": {"reason"}, "admin.disable": {"reason"}, "admin.product.activate": {"reason"}, "admin.product.suspend": {"reason"}, "admin.event.retry": {"reason"},
}
var Required = map[string][]string{
	"auth.request": {"phone"}, "auth.verify": {"challenge_id", "code"}, "auth.mfa": {"challenge_id", "code"}, "auth.refresh": {"refresh_token"},
	"address.create": {"label", "address_line", "city", "pincode"},
	"order.create":   {"product_id", "service_date", "quantity", "delivery_address"}, "seller.product.create": {"name", "price", "daily_capacity"}, "seller.product.availability": {"is_available"},
	"seller.customer.create": {"full_name", "phone", "delivery_address"}, "seller.customer.price.set": {"product_id", "price"}, "seller.customer.price.delete": {"product_id"}, "seller.delivery.create": {"customer_id", "product_id", "service_date", "quantity"},
	"order.cancel": {"reason"}, "seller.order.result": {"result"}, "rating.set": {"rating"}, "extra.request.create": {"extra_quantity"}, "admin.order.correct": {"result", "reason"}, "admin.enable": {"reason"}, "admin.disable": {"reason"}, "admin.product.activate": {"reason"}, "admin.product.suspend": {"reason"}, "admin.event.retry": {"reason"},
}
