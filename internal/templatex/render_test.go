package templatex

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderEscapesJSON(t *testing.T) {
	raw := json.RawMessage(`{"text":"{{ get \"body.message\" }}","short":"{{ truncate (get \"body.message\") 4 }}"}`)
	out, err := Render(raw, map[string]any{"body": map[string]any{"message": "a\"b\nlong"}})
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]string
	if err = json.Unmarshal(out, &v); err != nil {
		t.Fatal(err)
	}
	if v["text"] != "a\"b\nlong" || !strings.HasSuffix(v["short"], "…") {
		t.Fatal(v)
	}
}
func TestMissingKeyFails(t *testing.T) {
	_, err := Render(json.RawMessage(`{"text":"{{ .body.missing }}"}`), map[string]any{"body": map[string]any{}})
	if err == nil {
		t.Fatal("expected missing key error")
	}
}
