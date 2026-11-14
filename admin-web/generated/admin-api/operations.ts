// Generated. Run make contracts.
export const operations = [
  {
    "id": "admin_audit",
    "path": "/v1/admin/audit-logs",
    "method": "GET"
  },
  {
    "id": "logout",
    "path": "/v1/admin/auth/logout",
    "method": "POST"
  },
  {
    "id": "auth_mfa",
    "path": "/v1/admin/auth/mfa/verify",
    "method": "POST"
  },
  {
    "id": "auth_request",
    "path": "/v1/admin/auth/otp/request",
    "method": "POST"
  },
  {
    "id": "auth_verify",
    "path": "/v1/admin/auth/otp/verify",
    "method": "POST"
  },
  {
    "id": "auth_refresh",
    "path": "/v1/admin/auth/token/refresh",
    "method": "POST"
  },
  {
    "id": "admin_summary",
    "path": "/v1/admin/bill-summary",
    "method": "GET"
  },
  {
    "id": "admin_dashboard",
    "path": "/v1/admin/dashboard",
    "method": "GET"
  },
  {
    "id": "admin_events",
    "path": "/v1/admin/events",
    "method": "GET"
  },
  {
    "id": "admin_event",
    "path": "/v1/admin/events/{id}",
    "method": "GET"
  },
  {
    "id": "admin_event_retry",
    "path": "/v1/admin/events/{id}/retry",
    "method": "POST"
  },
  {
    "id": "admin_me",
    "path": "/v1/admin/me",
    "method": "GET"
  },
  {
    "id": "admin_orders",
    "path": "/v1/admin/orders",
    "method": "GET"
  },
  {
    "id": "admin_order",
    "path": "/v1/admin/orders/{id}",
    "method": "GET"
  },
  {
    "id": "admin_order_correct",
    "path": "/v1/admin/orders/{id}/correct-delivery-result",
    "method": "POST"
  },
  {
    "id": "admin_products",
    "path": "/v1/admin/products",
    "method": "GET"
  },
  {
    "id": "admin_product",
    "path": "/v1/admin/products/{id}",
    "method": "GET"
  },
  {
    "id": "admin_product_activate",
    "path": "/v1/admin/products/{id}/activate",
    "method": "POST"
  },
  {
    "id": "admin_product_suspend",
    "path": "/v1/admin/products/{id}/suspend",
    "method": "POST"
  },
  {
    "id": "admin_users",
    "path": "/v1/admin/users",
    "method": "GET"
  },
  {
    "id": "admin_user",
    "path": "/v1/admin/users/{id}",
    "method": "GET"
  },
  {
    "id": "admin_disable",
    "path": "/v1/admin/users/{id}/disable",
    "method": "POST"
  },
  {
    "id": "admin_enable",
    "path": "/v1/admin/users/{id}/enable",
    "method": "POST"
  }
] as const;
export type OperationId=typeof operations[number]["id"];
