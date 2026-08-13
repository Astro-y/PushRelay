package templatex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"text/template"
	"time"

	"pushrelay/internal/rules"
)

func Render(raw json.RawMessage, data map[string]any) (json.RawMessage, error) {
	var value any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return nil, fmt.Errorf("template JSON: %w", err)
	}
	rendered, err := walk(value, data)
	if err != nil {
		return nil, err
	}
	out, err := json.Marshal(rendered)
	return out, err
}

func walk(value any, data map[string]any) (any, error) {
	switch v := value.(type) {
	case string:
		return renderString(v, data)
	case []any:
		out := make([]any, len(v))
		for i := range v {
			x, err := walk(v[i], data)
			if err != nil {
				return nil, err
			}
			out[i] = x
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, item := range v {
			x, err := walk(item, data)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w", k, err)
			}
			out[k] = x
		}
		return out, nil
	default:
		return value, nil
	}
}

func renderString(src string, data map[string]any) (string, error) {
	funcs := template.FuncMap{
		"get": func(path string) any { v, _ := rules.Get(data, path); return v },
		"default": func(fallback, value any) any {
			if value == nil || fmt.Sprint(value) == "" {
				return fallback
			}
			return value
		},
		"toJSON": func(value any) string { b, _ := json.Marshal(value); return string(b) },
		"truncate": func(value any, n int) string {
			s := fmt.Sprint(value)
			r := []rune(s)
			if len(r) <= n {
				return s
			}
			return string(r[:n]) + "…"
		},
		"dateInZone": func(layout, zone string, value any) string {
			loc, err := time.LoadLocation(zone)
			if err != nil {
				return ""
			}
			var t time.Time
			switch x := value.(type) {
			case float64:
				t = time.Unix(int64(x), 0)
			case json.Number:
				i, _ := x.Int64()
				t = time.Unix(i, 0)
			case string:
				t, _ = time.Parse(time.RFC3339, x)
			default:
				t = time.Now()
			}
			return t.In(loc).Format(layout)
		},
		"urlquery": func(value any) string { return url.QueryEscape(fmt.Sprint(value)) },
	}
	t, err := template.New("field").Option("missingkey=error").Funcs(funcs).Parse(src)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err = t.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}
