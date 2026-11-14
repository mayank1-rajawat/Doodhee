package domain

import (
	"fmt"
	"math/big"
	"strings"
)

// Record is a JSON transport-neutral entity. Monetary fields are decimal strings.
type Record map[string]any

func (r Record) Text(k string) string {
	if r[k] == nil {
		return ""
	}
	return fmt.Sprint(r[k])
}
func (r Record) Bool(k string) bool { b, _ := r[k].(bool); return b }

type Actor struct {
	ID, Role, Approval, Session, Audience string
	Enabled                               bool
}
type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string                { return e.Message }
func Fail(status int, code, msg string) error { return &Error{status, code, msg} }
func Require(a Actor, role string) error {
	if !a.Enabled {
		return Fail(403, "ACCOUNT_DISABLED", "This account is disabled.")
	}
	if a.Role != role {
		return Fail(403, "ROLE_FORBIDDEN", "This operation is not available for your role.")
	}
	return nil
}
func Transition(current, next, reason string, admin bool) error {
	if next == "not_delivered" {
		valid := map[string]bool{"customer_unavailable": true, "seller_unavailable": true, "insufficient_milk": true, "address_issue": true, "route_or_weather_issue": true, "cancelled_before_delivery": true, "other": true}
		if !valid[reason] {
			return Fail(422, "ORDER_REASON_REQUIRED", "Select a valid non-delivery reason.")
		}
	}
	if admin && (next == "delivered" || next == "not_delivered") {
		return nil
	}
	allowed := map[string][]string{"pending": {"accepted", "cancelled"}, "accepted": {"out_for_delivery", "cancelled"}, "out_for_delivery": {"delivered", "not_delivered"}}
	for _, s := range allowed[current] {
		if s == next {
			return nil
		}
	}
	return Fail(409, "ORDER_INVALID_TRANSITION", "The order status has changed. Refresh and try again.")
}
func Decimal(s string, scale int) (*big.Rat, error) {
	if len(s) > 16 || strings.ContainsAny(s, "eE/+- ") {
		return nil, Fail(422, "VALIDATION_ERROR", "Enter a valid positive decimal.")
	}
	parts := strings.Split(s, ".")
	maxDigits := 10
	if scale == 3 {
		maxDigits = 7
	}
	if len(parts[0]) == 0 || len(parts[0]) > maxDigits {
		return nil, Fail(422, "VALIDATION_ERROR", "The decimal value is outside the supported range.")
	}
	if len(parts) > 2 || (len(parts) == 2 && len(parts[1]) > scale) {
		return nil, Fail(422, "VALIDATION_ERROR", "Too many decimal places.")
	}
	v, ok := new(big.Rat).SetString(s)
	if !ok || v.Sign() <= 0 {
		return nil, Fail(422, "VALIDATION_ERROR", "Amount and quantity must be positive.")
	}
	return v, nil
}
func Total(q, p string) (string, error) {
	a, e := Decimal(q, 3)
	if e != nil {
		return "", e
	}
	b, e := Decimal(p, 2)
	if e != nil {
		return "", e
	}
	result := new(big.Rat).Mul(a, b).FloatString(2)
	if len(strings.Split(result, ".")[0]) > 10 {
		return "", Fail(422, "VALIDATION_ERROR", "The order total is too large.")
	}
	return result, nil
}
