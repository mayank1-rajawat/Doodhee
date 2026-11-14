// Generated. Run make contracts.
export interface Address {
  address_line?: string;
  city?: string;
  created_at?: string | null;
  id?: string;
  is_default?: boolean;
  label?: string;
  pincode?: string;
  updated_at?: string | null;
  user_id?: string;
}
export interface Audit {
  action?: string;
  admin_id?: string;
  after_data?: Record<string, unknown>;
  before_data?: Record<string, unknown>;
  created_at?: string | null;
  id?: string;
  reason?: string;
  request_id?: string;
  target_id?: string;
  target_type?: string;
}
export interface Challenge {
  challenge_id?: string;
  expires_in?: number;
  testing_otp?: string;
}
export interface CustomerLookup {
  already_connected?: boolean;
  delivery_address?: string;
  exists?: boolean;
  full_name?: string;
  id?: string;
  phone?: string;
  phone_verified_at?: string | null;
}
export interface CustomerPrice {
  created_at?: string | null;
  customer_id?: string;
  id?: string;
  milkman_id?: string;
  price?: string;
  product_id?: string;
  updated_at?: string | null;
}
export interface Event {
  aggregate_id?: string;
  aggregate_type?: string;
  attempt_count?: number;
  created_at?: string | null;
  event_type?: string;
  id?: string;
  last_error?: string;
  payload?: Record<string, unknown>;
  published_at?: string | null;
  status?: string;
}
export interface ExtraMilkRequest {
  base_quantity?: string;
  created_at?: string | null;
  customer_id?: string;
  decided_at?: string | null;
  decision_note?: string;
  extra_quantity?: string;
  full_name?: string;
  id?: string;
  milkman_id?: string;
  order_id?: string;
  phone?: string;
  planned_quantity?: string;
  product_name?: string;
  service_date?: string;
  status?: "pending" | "approved" | "rejected" | "none";
  updated_at?: string | null;
}
export interface MFAChallenge {
  mfa_challenge_id?: string;
}
export interface MilkmanCustomer {
  created_at?: string | null;
  delivery_address?: string;
  full_name?: string;
  id?: string;
  is_active?: boolean;
  phone?: string;
  phone_verified_at?: string | null;
  relationship_id?: string;
  updated_at?: string | null;
}
export interface MilkmanRating {
  created_at?: string | null;
  customer_id?: string;
  id?: string;
  milkman_id?: string;
  rating?: number | null;
  updated_at?: string | null;
}
export interface Order {
  approved_extra_quantity?: string;
  base_quantity?: string;
  created_at?: string | null;
  customer_id?: string;
  customer_note?: string;
  delivery_address?: string;
  delivery_marked_at?: string | null;
  delivery_note?: string;
  id?: string;
  marked_by?: string;
  not_delivered_reason?: string | null;
  price_source?: "default" | "customer" | "delivery";
  product_id?: string;
  quantity?: string;
  seller_id?: string;
  service_date?: string;
  status?: "pending" | "accepted" | "out_for_delivery" | "delivered" | "not_delivered" | "cancelled";
  total_amount?: string;
  unit_price?: string;
  updated_at?: string | null;
}
export interface Product {
  admin_suspended?: boolean;
  created_at?: string | null;
  daily_capacity?: string;
  id?: string;
  is_available?: boolean;
  name?: string;
  price?: string;
  seller_id?: string;
  unit?: string;
  updated_at?: string | null;
}
export interface PublicSeller {
  average_rating?: string;
  city?: string;
  full_name?: string;
  id?: string;
  pincode?: string;
  rank?: number;
  rating_count?: number;
}
export interface Summary {
  delivered_days?: number;
  delivered_litres?: string;
  groups?: Record<string, unknown>;
  total_amount?: string;
}
export interface Tokens {
  access_token?: string;
  refresh_token?: string;
  user?: User;
}
export interface User {
  address_line?: string;
  approval_reason?: string;
  approval_status?: "pending" | "approved" | "rejected";
  approved_at?: string | null;
  approved_by?: string;
  city?: string;
  created_at?: string | null;
  full_name?: string;
  id?: string;
  is_enabled?: boolean;
  phone?: string;
  phone_verified_at?: string | null;
  pincode?: string;
  role?: "customer" | "seller" | "admin";
  updated_at?: string | null;
}