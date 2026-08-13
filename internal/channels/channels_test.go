package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestAllChannelAdapters(t *testing.T) {
	var mu sync.Mutex
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests[r.URL.Path]++
		mu.Unlock()
		if strings.Contains(r.URL.Path, "gettoken") {
			_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 0, "access_token": "access"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errcode":0,"code":0,"ok":true}`)
	}))
	defer server.Close()
	s := New(true)
	s.telegramBase = server.URL
	s.wecomBase = server.URL
	tests := []struct {
		name, typ string
		config    map[string]any
		body      string
	}{
		{"dingtalk", "dingtalk", map[string]any{"webhook": server.URL + "/dingtalk", "secret": "sign"}, `{"msgtype":"text","text":{"content":"hello"}}`},
		{"wecom", "wecom", map[string]any{"webhook": server.URL + "/wecom"}, `{"msgtype":"text","text":{"content":"hello"}}`},
		{"wecom_app", "wecom_app", map[string]any{"corp_id": "corp", "agent_id": "1", "secret": "secret"}, `{"msgtype":"text","text":{"content":"hello"}}`},
		{"feishu", "feishu", map[string]any{"webhook": server.URL + "/feishu", "secret": "sign"}, `{"msg_type":"text","content":{"text":"hello"}}`},
		{"telegram", "telegram", map[string]any{"bot_token": "token", "chat_id": "42"}, `{"text":"hello"}`},
		{"discord", "discord", map[string]any{"webhook": server.URL + "/discord"}, `{"content":"hello"}`},
		{"bark", "bark", map[string]any{"webhook": server.URL + "/bark"}, `{"title":"test","body":"hello"}`},
		{"webhook", "webhook", map[string]any{"webhook": server.URL + "/generic"}, `{"method":"POST","headers":{"X-Test":"yes"},"body":{"message":"hello"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, excerpt, err := s.Send(context.Background(), tt.typ, tt.config, json.RawMessage(tt.body))
			if err != nil {
				t.Fatalf("send: %v (%s)", err, excerpt)
			}
			if status != http.StatusOK {
				t.Fatalf("status=%d", status)
			}
		})
	}
	mu.Lock()
	defer mu.Unlock()
	for _, path := range []string{"/dingtalk", "/wecom", "/cgi-bin/gettoken", "/cgi-bin/message/send", "/feishu", "/bottoken/sendMessage", "/discord", "/bark", "/generic"} {
		found := false
		for actual, count := range requests {
			if strings.HasPrefix(actual, path) && count > 0 {
				found = true
			}
		}
		if !found {
			t.Errorf("expected request path %s, got %v", path, requests)
		}
	}
}

func TestPlatformErrorIsReturned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"errcode":310000,"errmsg":"bad signature"}`)
	}))
	defer server.Close()
	s := New(true)
	_, _, err := s.Send(context.Background(), "dingtalk", map[string]any{"webhook": server.URL}, json.RawMessage(`{"msgtype":"text","text":{"content":"hello"}}`))
	if err == nil {
		t.Fatal("expected platform error")
	}
}

func TestPrivateWebhookBlocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()
	s := New(false)
	_, _, err := s.Send(context.Background(), "webhook", map[string]any{"webhook": server.URL}, json.RawMessage(`{"method":"POST","body":{}}`))
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("expected SSRF block, got %v", err)
	}
}

func TestPrivateWebhookSettingUpdatesAtRuntime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	s := New(false)
	payload := json.RawMessage(`{"method":"POST","body":{}}`)
	if _, _, err := s.Send(context.Background(), "webhook", map[string]any{"webhook": server.URL}, payload); err == nil {
		t.Fatal("expected private target to be blocked")
	}
	s.SetAllowPrivate(true)
	status, _, err := s.Send(context.Background(), "webhook", map[string]any{"webhook": server.URL}, payload)
	if err != nil || status != http.StatusNoContent {
		t.Fatalf("runtime update did not take effect: status=%d err=%v", status, err)
	}
}
