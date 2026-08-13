package rules

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Node struct {
	Op       string `json:"op,omitempty"`
	Path     string `json:"path,omitempty"`
	Operator string `json:"operator,omitempty"`
	Value    any    `json:"value,omitempty"`
	Children []Node `json:"children,omitempty"`
}

func Parse(raw json.RawMessage) (Node, error) {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return Node{Op: "and"}, nil
	}
	var n Node
	if err := json.Unmarshal(raw, &n); err != nil {
		return Node{}, err
	}
	if err := validate(n, 0); err != nil {
		return Node{}, err
	}
	return n, nil
}

func validate(n Node, depth int) error {
	if depth > 5 {
		return fmt.Errorf("condition tree exceeds maximum depth")
	}
	if n.Op != "" {
		if n.Op != "and" && n.Op != "or" {
			return fmt.Errorf("unsupported group operator %q", n.Op)
		}
		if len(n.Children) > 50 {
			return fmt.Errorf("condition group is too large")
		}
		for _, c := range n.Children {
			if err := validate(c, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	switch n.Operator {
	case "exists", "not_exists", "eq", "ne", "contains", "not_contains", "regex", "gt", "gte", "lt", "lte", "in", "not_in":
	default:
		return fmt.Errorf("unsupported operator %q", n.Operator)
	}
	if strings.TrimSpace(n.Path) == "" {
		return fmt.Errorf("condition path is required")
	}
	return nil
}

func Match(n Node, data map[string]any) (bool, error) {
	if n.Op != "" {
		if len(n.Children) == 0 {
			return true, nil
		}
		if n.Op == "and" {
			for _, c := range n.Children {
				ok, err := Match(c, data)
				if err != nil || !ok {
					return ok, err
				}
			}
			return true, nil
		}
		for _, c := range n.Children {
			ok, err := Match(c, data)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	got, exists := Get(data, n.Path)
	switch n.Operator {
	case "exists":
		return exists, nil
	case "not_exists":
		return !exists, nil
	}
	if !exists {
		return false, nil
	}
	switch n.Operator {
	case "eq":
		return equal(got, n.Value), nil
	case "ne":
		return !equal(got, n.Value), nil
	case "contains", "not_contains":
		ok := strings.Contains(fmt.Sprint(got), fmt.Sprint(n.Value))
		if n.Operator == "not_contains" {
			ok = !ok
		}
		return ok, nil
	case "regex":
		re, err := regexp.Compile(fmt.Sprint(n.Value))
		if err != nil {
			return false, err
		}
		return re.MatchString(fmt.Sprint(got)), nil
	case "gt", "gte", "lt", "lte":
		a, ok1 := number(got)
		b, ok2 := number(n.Value)
		if !ok1 || !ok2 {
			return false, nil
		}
		switch n.Operator {
		case "gt":
			return a > b, nil
		case "gte":
			return a >= b, nil
		case "lt":
			return a < b, nil
		default:
			return a <= b, nil
		}
	case "in", "not_in":
		rv := reflect.ValueOf(n.Value)
		ok := false
		if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) {
			for i := 0; i < rv.Len(); i++ {
				if equal(got, rv.Index(i).Interface()) {
					ok = true
					break
				}
			}
		}
		if n.Operator == "not_in" {
			ok = !ok
		}
		return ok, nil
	}
	return false, nil
}

func Get(data any, path string) (any, bool) {
	path = strings.ReplaceAll(path, "[", ".")
	path = strings.ReplaceAll(path, "]", "")
	parts := strings.Split(strings.Trim(path, "."), ".")
	cur := data
	for _, part := range parts {
		switch v := cur.(type) {
		case map[string]any:
			var ok bool
			cur, ok = v[part]
			if !ok {
				return nil, false
			}
		case []any:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(v) {
				return nil, false
			}
			cur = v[i]
		default:
			return nil, false
		}
	}
	return cur, true
}
func equal(a, b any) bool {
	if na, ok := number(a); ok {
		if nb, ok2 := number(b); ok2 {
			return na == nb
		}
	}
	return reflect.DeepEqual(a, b) || fmt.Sprint(a) == fmt.Sprint(b)
}
func number(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, e := n.Float64()
		return f, e == nil
	case string:
		f, e := strconv.ParseFloat(n, 64)
		return f, e == nil
	}
	return 0, false
}

var sensitive = regexp.MustCompile(`(?i)(authorization|cookie|token|password|passwd|secret|api[_-]?key|access[_-]?key)`)

func Redact(v any) any {
	return RedactWithFields(v, nil)
}

func RedactWithFields(v any, fields []string) any {
	custom := make(map[string]bool, len(fields))
	for _, field := range fields {
		custom[strings.ToLower(strings.TrimSpace(field))] = true
	}
	return redactValue(v, custom)
}

func redactValue(v any, custom map[string]bool) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			if sensitive.MatchString(k) || custom[strings.ToLower(k)] {
				out[k] = "[REDACTED]"
			} else {
				out[k] = redactValue(val, custom)
			}
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = redactValue(x[i], custom)
		}
		return out
	default:
		return v
	}
}
