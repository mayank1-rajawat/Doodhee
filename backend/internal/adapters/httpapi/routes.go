package httpapi

type Route struct{ Method, Path, Op string }

var Public = []Route{
	{"POST", "/v1/auth/otp/request", "auth.request"}, {"POST", "/v1/auth/otp/verify", "auth.verify"}, {"POST", "/v1/auth/token/refresh", "auth.refresh"}, {"POST", "/v1/auth/logout", "logout"}, {"POST", "/v1/auth/logout-all", "logout.all"},
	{"GET", "/v1/me", "me"}, {"PATCH", "/v1/me", "profile"},
	{"GET", "/v1/addresses", "addresses"}, {"POST", "/v1/addresses", "address.create"}, {"PATCH", "/v1/addresses/{id}", "address.update"}, {"DELETE", "/v1/addresses/{id}", "address.delete"}, {"POST", "/v1/addresses/{id}/default", "address.default"},
	{"GET", "/v1/sellers", "sellers"}, {"GET", "/v1/sellers/rankings", "rankings"}, {"GET", "/v1/sellers/{id}", "discovery.detail"}, {"GET", "/v1/sellers/{id}/products", "products"}, {"GET", "/v1/sellers/{id}/rating", "rating.get"}, {"PUT", "/v1/sellers/{id}/rating", "rating.set"}, {"GET", "/v1/products/{id}", "product"},
	{"POST", "/v1/orders", "order.create"}, {"GET", "/v1/orders", "orders"}, {"GET", "/v1/orders/{id}", "order"}, {"POST", "/v1/orders/{id}/cancel", "order.cancel"}, {"POST", "/v1/orders/{id}/extra-milk-request", "extra.request.create"}, {"GET", "/v1/orders/{id}/extra-milk-request", "extra.request.get"}, {"GET", "/v1/bill-summary", "summary"},
	{"GET", "/v1/seller/approval-status", "seller.approval"}, {"GET", "/v1/seller/profile", "seller.profile"}, {"PATCH", "/v1/seller/profile", "seller.profile.update"},
	{"GET", "/v1/seller/products", "seller.products"}, {"POST", "/v1/seller/products", "seller.product.create"}, {"GET", "/v1/seller/products/{id}", "seller.product"}, {"PATCH", "/v1/seller/products/{id}", "seller.product.update"}, {"POST", "/v1/seller/products/{id}/availability", "seller.product.availability"},
	{"GET", "/v1/seller/customers", "seller.customers"}, {"POST", "/v1/seller/customers", "seller.customer.create"}, {"GET", "/v1/seller/customers/lookup", "seller.customer.lookup"}, {"GET", "/v1/seller/customers/{id}", "seller.customer"}, {"GET", "/v1/seller/customers/{id}/prices", "seller.customer.prices"}, {"PUT", "/v1/seller/customers/{id}/price", "seller.customer.price.set"}, {"DELETE", "/v1/seller/customers/{id}/price", "seller.customer.price.delete"},
	{"POST", "/v1/seller/deliveries", "seller.delivery.create"}, {"GET", "/v1/seller/orders", "seller.orders"}, {"GET", "/v1/seller/orders/{id}", "seller.order"}, {"POST", "/v1/seller/orders/{id}/accept", "seller.order.accept"}, {"POST", "/v1/seller/orders/{id}/out-for-delivery", "seller.order.dispatch"}, {"POST", "/v1/seller/orders/{id}/delivery-result", "seller.order.result"}, {"GET", "/v1/seller/extra-milk-requests", "seller.extra.requests"}, {"POST", "/v1/seller/extra-milk-requests/{id}/approve", "seller.extra.approve"}, {"POST", "/v1/seller/extra-milk-requests/{id}/reject", "seller.extra.reject"}, {"GET", "/v1/seller/bill-summary", "seller.summary"},
}
var Admin = []Route{
	{"POST", "/v1/admin/auth/otp/request", "auth.request"}, {"POST", "/v1/admin/auth/otp/verify", "auth.verify"}, {"POST", "/v1/admin/auth/mfa/verify", "auth.mfa"}, {"POST", "/v1/admin/auth/token/refresh", "auth.refresh"}, {"POST", "/v1/admin/auth/logout", "logout"}, {"GET", "/v1/admin/me", "admin.me"},
	{"GET", "/v1/admin/users", "admin.users"}, {"GET", "/v1/admin/users/{id}", "admin.user"}, {"POST", "/v1/admin/users/{id}/enable", "admin.enable"}, {"POST", "/v1/admin/users/{id}/disable", "admin.disable"},
	{"GET", "/v1/admin/products", "admin.products"}, {"GET", "/v1/admin/products/{id}", "admin.product"}, {"POST", "/v1/admin/products/{id}/suspend", "admin.product.suspend"}, {"POST", "/v1/admin/products/{id}/activate", "admin.product.activate"},
	{"GET", "/v1/admin/orders", "admin.orders"}, {"GET", "/v1/admin/orders/{id}", "admin.order"}, {"POST", "/v1/admin/orders/{id}/correct-delivery-result", "admin.order.correct"}, {"GET", "/v1/admin/bill-summary", "admin.summary"}, {"GET", "/v1/admin/dashboard", "admin.dashboard"},
	{"GET", "/v1/admin/events", "admin.events"}, {"GET", "/v1/admin/events/{id}", "admin.event"}, {"POST", "/v1/admin/events/{id}/retry", "admin.event.retry"}, {"GET", "/v1/admin/audit-logs", "admin.audit"},
}
