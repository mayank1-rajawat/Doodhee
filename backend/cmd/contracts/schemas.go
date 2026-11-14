package main

import "strings"

type obj = map[string]any

func str() obj             { return obj{"type": "string"} }
func ref(name string) obj  { return obj{"$ref": "#/components/schemas/" + name} }
func object(props obj) obj { return obj{"type": "object", "properties": props} }
func paged(item obj) obj {
	return object(obj{"items": obj{"type": "array", "items": item}, "next_cursor": str()})
}
func schemas() obj {
	decimal := obj{"type": "string", "pattern": "^[0-9]+(\\.[0-9]+)?$"}
	uuid := obj{"type": "string", "format": "uuid"}
	stamp := obj{"type": "string", "format": "date-time", "nullable": true}
	boolean := obj{"type": "boolean"}
	user := obj{"id": uuid, "phone": str(), "full_name": str(), "role": obj{"type": "string", "enum": []string{"customer", "seller", "admin"}}, "approval_status": obj{"type": "string", "enum": []string{"pending", "approved", "rejected"}}, "is_enabled": boolean, "address_line": str(), "city": str(), "pincode": str(), "approval_reason": str(), "approved_by": uuid, "approved_at": stamp, "phone_verified_at": stamp, "created_at": stamp, "updated_at": stamp}
	address := obj{"id": uuid, "user_id": uuid, "label": str(), "address_line": str(), "city": str(), "pincode": str(), "is_default": boolean, "created_at": stamp, "updated_at": stamp}
	product := obj{"id": uuid, "seller_id": uuid, "name": str(), "unit": str(), "price": decimal, "daily_capacity": decimal, "is_available": boolean, "admin_suspended": boolean, "created_at": stamp, "updated_at": stamp}
	order := obj{"id": uuid, "customer_id": uuid, "seller_id": uuid, "product_id": uuid, "service_date": obj{"type": "string", "format": "date"}, "quantity": decimal, "base_quantity": decimal, "approved_extra_quantity": decimal, "unit_price": decimal, "price_source": obj{"type": "string", "enum": []string{"default", "customer", "delivery"}}, "total_amount": decimal, "status": obj{"type": "string", "enum": []string{"pending", "accepted", "out_for_delivery", "delivered", "not_delivered", "cancelled"}}, "not_delivered_reason": obj{"type": "string", "nullable": true}, "delivery_note": str(), "delivery_address": str(), "customer_note": str(), "marked_by": uuid, "delivery_marked_at": stamp, "created_at": stamp, "updated_at": stamp}
	milkmanCustomer := obj{"id": uuid, "full_name": str(), "phone": str(), "phone_verified_at": stamp, "relationship_id": uuid, "delivery_address": str(), "is_active": boolean, "created_at": stamp, "updated_at": stamp}
	customerLookup := obj{"exists": boolean, "id": uuid, "full_name": str(), "phone": str(), "phone_verified_at": stamp, "already_connected": boolean, "delivery_address": str()}
	customerPrice := obj{"id": uuid, "milkman_id": uuid, "customer_id": uuid, "product_id": uuid, "price": decimal, "created_at": stamp, "updated_at": stamp}
	extraRequest := obj{"id": uuid, "order_id": uuid, "customer_id": uuid, "milkman_id": uuid, "extra_quantity": decimal, "status": obj{"type": "string", "enum": []string{"pending", "approved", "rejected", "none"}}, "decision_note": str(), "decided_at": stamp, "service_date": obj{"type": "string", "format": "date"}, "planned_quantity": decimal, "base_quantity": decimal, "full_name": str(), "phone": str(), "product_name": str(), "created_at": stamp, "updated_at": stamp}
	rating := obj{"id": uuid, "customer_id": uuid, "milkman_id": uuid, "rating": obj{"type": "integer", "minimum": 1, "maximum": 5, "nullable": true}, "created_at": stamp, "updated_at": stamp}
	publicSeller := obj{"id": uuid, "full_name": str(), "city": str(), "pincode": str(), "average_rating": decimal, "rating_count": obj{"type": "integer"}, "rank": obj{"type": "integer"}}
	return obj{"User": object(user), "Address": object(address), "Product": object(product), "Order": object(order), "MilkmanCustomer": object(milkmanCustomer), "CustomerLookup": object(customerLookup), "CustomerPrice": object(customerPrice), "ExtraMilkRequest": object(extraRequest), "MilkmanRating": object(rating), "PublicSeller": object(publicSeller), "Summary": object(obj{"delivered_days": obj{"type": "integer"}, "delivered_litres": decimal, "total_amount": decimal, "groups": obj{"type": "object", "additionalProperties": object(obj{"user_id": uuid, "delivered_litres": decimal, "total_amount": decimal})}}), "Tokens": object(obj{"access_token": str(), "refresh_token": str(), "user": ref("User")}), "Challenge": object(obj{"challenge_id": uuid, "expires_in": obj{"type": "integer"}, "testing_otp": obj{"type": "string", "pattern": "^[0-9]{6}$", "description": "Present only when TESTING_MODE is enabled in development or test."}}), "MFAChallenge": object(obj{"mfa_challenge_id": uuid}), "Event": object(obj{"id": uuid, "event_type": str(), "aggregate_type": str(), "aggregate_id": uuid, "status": str(), "payload": obj{"type": "object"}, "attempt_count": obj{"type": "integer"}, "last_error": str(), "created_at": stamp, "published_at": stamp}), "Audit": object(obj{"id": uuid, "admin_id": uuid, "action": str(), "target_type": str(), "target_id": uuid, "reason": str(), "request_id": str(), "before_data": obj{"type": "object"}, "after_data": obj{"type": "object"}, "created_at": stamp})}
}
func responseSchema(op string) obj {
	switch op {
	case "auth.request":
		return ref("Challenge")
	case "auth.verify":
		return obj{"oneOf": []any{ref("Tokens"), ref("MFAChallenge")}}
	case "auth.refresh", "auth.mfa":
		return ref("Tokens")
	case "me", "profile", "seller.profile", "seller.profile.update", "seller.approval", "admin.me", "admin.user", "admin.enable", "admin.disable":
		return ref("User")
	case "addresses":
		return paged(ref("Address"))
	case "address.create", "address.update", "address.delete", "address.default":
		return ref("Address")
	case "sellers", "rankings":
		return paged(ref("PublicSeller"))
	case "discovery.detail":
		return ref("PublicSeller")
	case "products", "seller.products", "admin.products":
		return paged(ref("Product"))
	case "product", "seller.product", "seller.product.create", "seller.product.update", "seller.product.availability", "admin.product", "admin.product.suspend", "admin.product.activate":
		return ref("Product")
	case "orders", "seller.orders", "admin.orders":
		return paged(ref("Order"))
	case "seller.delivery.create":
		return ref("Order")
	case "seller.customers":
		return paged(ref("MilkmanCustomer"))
	case "seller.customer.lookup":
		return ref("CustomerLookup")
	case "seller.customer", "seller.customer.create":
		return ref("MilkmanCustomer")
	case "seller.customer.prices":
		return paged(ref("CustomerPrice"))
	case "seller.customer.price.set", "seller.customer.price.delete":
		return ref("CustomerPrice")
	case "rating.get", "rating.set":
		return ref("MilkmanRating")
	case "seller.extra.requests":
		return paged(ref("ExtraMilkRequest"))
	case "extra.request.create", "extra.request.get", "seller.extra.approve", "seller.extra.reject":
		return ref("ExtraMilkRequest")
	case "summary", "seller.summary", "admin.summary":
		return ref("Summary")
	case "admin.events":
		return paged(ref("Event"))
	case "admin.event", "admin.event.retry":
		return ref("Event")
	case "admin.audit":
		return paged(ref("Audit"))
	}
	if strings.Contains(op, "order") {
		return ref("Order")
	}
	return obj{"type": "object"}
}
