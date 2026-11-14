package main

import (
	contract "doodhee-backend/contracts/api"
	"doodhee-backend/internal/adapters/httpapi"
	"encoding/json"
	"os"
	"strings"
)

func main() {
	for name, routes := range map[string][]httpapi.Route{"public": httpapi.Public, "admin": httpapi.Admin} {
		paths := map[string]any{}
		for _, r := range routes {
			params := []any{}
			if strings.Contains(r.Path, "{id}") {
				params = append(params, map[string]any{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}})
			}
			op := map[string]any{"operationId": strings.ReplaceAll(r.Op, ".", "_"), "summary": r.Op, "parameters": params, "responses": map[string]any{"200": map[string]any{"description": "Success", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "properties": map[string]any{"data": map[string]any{}, "meta": map[string]any{"type": "object"}}}}}}, "default": map[string]any{"description": "Structured API error"}}}
			if r.Method != "GET" {
				properties := map[string]any{}
				for _, field := range contract.Fields[r.Op] {
					typ := "string"
					if field == "is_available" || field == "is_default" {
						typ = "boolean"
					}
					properties[field] = map[string]any{"type": typ}
				}
				inputSchema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
				if len(contract.Required[r.Op]) > 0 {
					inputSchema["required"] = contract.Required[r.Op]
				}
				op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": inputSchema}}}
			}
			status := "200"
			if r.Op == "order.create" || r.Op == "seller.product.create" || r.Op == "address.create" || r.Op == "seller.customer.create" || r.Op == "seller.delivery.create" || r.Op == "extra.request.create" {
				status = "201"
			}
			if strings.HasPrefix(r.Op, "logout") {
				status = "204"
			}
			result := map[string]any{"description": "Success"}
			if status != "204" {
				result["content"] = obj{"application/json": obj{"schema": object(obj{"data": responseSchema(r.Op), "meta": object(obj{"request_id": str()})})}}
			}
			op["responses"] = obj{status: result, "default": obj{"description": "Structured API error", "content": obj{"application/json": obj{"schema": object(obj{"error": object(obj{"code": str(), "message": str(), "fields": obj{"type": "object"}}), "meta": object(obj{"request_id": str()})})}}}}
			if r.Method == "GET" {
				for _, q := range []string{"from", "to", "cursor", "status", "seller_id", "customer_id", "service_date", "role", "approval_status", "is_enabled", "is_available", "phone", "pincode", "event_type", "admin_id", "action", "target_id"} {
					params = append(params, map[string]any{"name": q, "in": "query", "schema": map[string]any{"type": "string"}})
				}
				op["parameters"] = params
			}
			if r.Op == "order.create" || r.Op == "admin.order.correct" || r.Op == "seller.order.result" || r.Op == "seller.delivery.create" || r.Op == "extra.request.create" || r.Op == "seller.extra.approve" || r.Op == "seller.extra.reject" {
				params = append(params, map[string]any{"name": "Idempotency-Key", "in": "header", "required": true, "schema": map[string]any{"type": "string", "minLength": 8}})
				op["parameters"] = params
			}
			if !strings.HasPrefix(r.Op, "auth.") {
				op["security"] = []any{map[string]any{"bearerAuth": []any{}}}
			}
			path, ok := paths[r.Path].(map[string]any)
			if !ok {
				path = map[string]any{}
			}
			path[strings.ToLower(r.Method)] = op
			paths[r.Path] = path
		}
		spec := map[string]any{"openapi": "3.0.3", "info": map[string]any{"title": "Doodhee " + name + " API", "version": "1.0.0"}, "paths": paths, "components": map[string]any{"schemas": schemas(), "securitySchemes": map[string]any{"bearerAuth": map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "JWT"}}}}
		b, _ := json.MarshalIndent(spec, "", "  ")
		os.MkdirAll("api/openapi", 0755)
		if e := os.WriteFile("api/openapi/"+name+"-v1.yaml", b, 0644); e != nil {
			panic(e)
		}
	}
}
