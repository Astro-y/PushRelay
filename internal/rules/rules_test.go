package rules

import (
	"encoding/json"
	"testing"
)

func TestNestedRulesAndOperators(t *testing.T) {
	data := map[string]any{"body": map[string]any{"status": "error", "latency": 250.0, "tags": []any{"prod", "api"}}}
	n := Node{Op: "and", Children: []Node{{Path: "body.status", Operator: "eq", Value: "error"}, {Op: "or", Children: []Node{{Path: "body.latency", Operator: "gt", Value: 200}, {Path: "body.status", Operator: "eq", Value: "ok"}}}}}
	ok, err := Match(n, data)
	if err != nil || !ok {
		t.Fatalf("expected match, got %v %v", ok, err)
	}
	if ok, _ = Match(Node{Path: "body.missing", Operator: "exists"}, data); ok {
		t.Fatal("missing path must not exist")
	}
}

func TestParseRejectsUnknownOperator(t *testing.T) {
	if _, err := Parse(json.RawMessage(`{"path":"body.x","operator":"eval","value":"x"}`)); err == nil {
		t.Fatal("expected validation error")
	}
}
func TestRedact(t *testing.T) {
	v := Redact(map[string]any{"password": "x", "nested": map[string]any{"api_key": "y", "safe": "z"}}).(map[string]any)
	if v["password"] != "[REDACTED]" {
		t.Fatal(v)
	}
	nested := v["nested"].(map[string]any)
	if nested["safe"] != "z" || nested["api_key"] != "[REDACTED]" {
		t.Fatal(nested)
	}
}

func TestCustomRedact(t *testing.T) {
	v := RedactWithFields(map[string]any{"customer_code": "C-1", "safe": "ok"}, []string{"CUSTOMER_CODE"}).(map[string]any)
	if v["customer_code"] != "[REDACTED]" || v["safe"] != "ok" {
		t.Fatal(v)
	}
}
