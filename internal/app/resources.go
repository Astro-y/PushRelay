package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pushrelay/internal/channels"
	"pushrelay/internal/rules"
	"pushrelay/internal/schedule"
	"pushrelay/internal/secure"
	"pushrelay/internal/store"
	"pushrelay/internal/templatex"
)

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	v, err := s.store.Dashboard(r.Context())
	if err != nil {
		dbError(w, r, err)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) channelTypes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, []map[string]any{
		{"type": "dingtalk", "label": "钉钉群机器人", "required": []string{"webhook"}, "optional": []string{"secret"}},
		{"type": "wecom", "label": "企业微信群机器人", "required": []string{"webhook"}},
		{"type": "wecom_app", "label": "企业微信应用", "required": []string{"corp_id", "agent_id", "secret"}},
		{"type": "feishu", "label": "飞书群机器人", "required": []string{"webhook"}, "optional": []string{"secret"}},
		{"type": "telegram", "label": "Telegram Bot", "required": []string{"bot_token", "chat_id"}},
		{"type": "discord", "label": "Discord Webhook", "required": []string{"webhook"}},
		{"type": "bark", "label": "Bark", "required": []string{"webhook"}},
		{"type": "webhook", "label": "通用 Webhook", "required": []string{"webhook"}},
	})
}

func (s *Server) listChannels(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListChannels(r.Context())
	if err != nil {
		dbError(w, r, err)
		return
	}
	for i := range items {
		items[i].Config = maskConfig(items[i].Type)
	}
	writeJSON(w, 200, items)
}
func (s *Server) saveChannel(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string         `json:"name"`
		Type    string         `json:"type"`
		Config  map[string]any `json:"config"`
		Enabled *bool          `json:"enabled"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := requireName(in.Name); err != nil {
		writeError(w, 400, "validation_error", err.Error(), nil, r)
		return
	}
	if err := channels.Validate(in.Type, in.Config); err != nil {
		writeError(w, 400, "validation_error", err.Error(), nil, r)
		return
	}
	raw, _ := json.Marshal(in.Config)
	enc, err := s.vault.Encrypt(raw)
	if err != nil {
		dbError(w, r, err)
		return
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	v, err := s.store.SaveChannel(r.Context(), store.Channel{ID: pathID(r), Name: strings.TrimSpace(in.Name), Type: in.Type, ConfigEnc: enc, Enabled: enabled})
	if err != nil {
		dbError(w, r, err)
		return
	}
	v.Config = maskConfig(v.Type)
	status := 200
	if pathID(r) == "" {
		status = 201
	}
	writeJSON(w, status, v)
}
func (s *Server) deleteChannel(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteChannel(r.Context(), pathID(r)); err != nil {
		dbError(w, r, err)
		return
	}
	w.WriteHeader(204)
}
func maskConfig(typ string) map[string]any {
	out := map[string]any{"configured": true}
	switch typ {
	case "wecom_app":
		out["fields"] = []string{"corp_id", "agent_id", "secret"}
	case "telegram":
		out["fields"] = []string{"bot_token", "chat_id"}
	default:
		out["fields"] = []string{"webhook", "secret"}
	}
	return out
}

func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request) {
	v, err := s.store.ListTemplates(r.Context())
	if err != nil {
		dbError(w, r, err)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) saveTemplate(w http.ResponseWriter, r *http.Request) {
	var v store.MessageTemplate
	if !decodeJSON(w, r, &v) {
		return
	}
	v.ID = pathID(r)
	if err := requireName(v.Name); err != nil {
		writeError(w, 400, "validation_error", err.Error(), nil, r)
		return
	}
	if !channels.Supported[v.ChannelType] {
		writeError(w, 400, "validation_error", "unsupported channel_type", nil, r)
		return
	}
	if !json.Valid(v.Body) {
		writeError(w, 400, "validation_error", "body must be valid JSON", nil, r)
		return
	}
	if _, err := templatex.Render(v.Body, sampleContext()); err != nil && !strings.Contains(err.Error(), "map has no entry") {
		writeError(w, 400, "template_error", err.Error(), nil, r)
		return
	}
	saved, err := s.store.SaveTemplate(r.Context(), v)
	if err != nil {
		dbError(w, r, err)
		return
	}
	status := 200
	if pathID(r) == "" {
		status = 201
	}
	writeJSON(w, status, saved)
}
func (s *Server) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteTemplate(r.Context(), pathID(r)); err != nil {
		dbError(w, r, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) preview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Body    json.RawMessage `json:"body"`
		Context map[string]any  `json:"context"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Context == nil {
		in.Context = sampleContext()
	}
	out, err := templatex.Render(in.Body, in.Context)
	if err != nil {
		writeError(w, 400, "template_error", err.Error(), nil, r)
		return
	}
	var value any
	_ = json.Unmarshal(out, &value)
	writeJSON(w, 200, map[string]any{"rendered": value})
}

func (s *Server) listGroups(w http.ResponseWriter, r *http.Request) {
	v, err := s.store.ListGroups(r.Context())
	if err != nil {
		dbError(w, r, err)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) saveGroup(w http.ResponseWriter, r *http.Request) {
	var v store.TargetGroup
	if !decodeJSON(w, r, &v) {
		return
	}
	v.ID = pathID(r)
	if err := requireName(v.Name); err != nil {
		writeError(w, 400, "validation_error", err.Error(), nil, r)
		return
	}
	saved, err := s.store.SaveGroup(r.Context(), v)
	if err != nil {
		dbError(w, r, err)
		return
	}
	status := 200
	if pathID(r) == "" {
		status = 201
	}
	writeJSON(w, status, saved)
}
func (s *Server) deleteGroup(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteGroup(r.Context(), pathID(r)); err != nil {
		dbError(w, r, err)
		return
	}
	w.WriteHeader(204)
}

type sourceInput struct {
	Name                  string   `json:"name"`
	HMACEnabled           bool     `json:"hmac_enabled"`
	AllowedCIDRs          []string `json:"allowed_cidrs"`
	CustomSensitiveFields []string `json:"custom_sensitive_fields"`
	MatchMode             string   `json:"match_mode"`
	PayloadPolicy         string   `json:"payload_policy"`
	Enabled               *bool    `json:"enabled"`
}

func (s *Server) listSources(w http.ResponseWriter, r *http.Request) {
	v, err := s.store.ListSources(r.Context())
	if err != nil {
		dbError(w, r, err)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) saveSource(w http.ResponseWriter, r *http.Request) {
	var in sourceInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := requireName(in.Name); err != nil {
		writeError(w, 400, "validation_error", err.Error(), nil, r)
		return
	}
	if in.MatchMode != "first_match" && in.MatchMode != "all_match" {
		writeError(w, 400, "validation_error", "match_mode must be first_match or all_match", nil, r)
		return
	}
	if in.PayloadPolicy != "redacted" && in.PayloadPolicy != "metadata" && in.PayloadPolicy != "none" {
		writeError(w, 400, "validation_error", "invalid payload_policy", nil, r)
		return
	}
	for _, raw := range in.AllowedCIDRs {
		if net.ParseIP(raw) == nil {
			if _, _, err := net.ParseCIDR(raw); err != nil {
				writeError(w, 400, "validation_error", "invalid IP or CIDR: "+raw, nil, r)
				return
			}
		}
	}
	if len(in.CustomSensitiveFields) > 50 {
		writeError(w, 400, "validation_error", "custom_sensitive_fields cannot contain more than 50 names", nil, r)
		return
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	id := pathID(r)
	var token, hmacSecret string
	var v store.Source
	var err error
	if id != "" {
		items, e := s.store.ListSources(r.Context())
		if e != nil {
			dbError(w, r, e)
			return
		}
		for _, item := range items {
			if item.ID == id {
				v = item
				break
			}
		}
		if v.ID == "" {
			writeError(w, 404, "not_found", "source not found", nil, r)
			return
		}
	} else {
		token, _ = secure.RandomToken(32)
		v.TokenHash = secure.HashToken(token)
		v.TokenPrefix = token[:8]
	}
	v.ID = id
	v.Name = strings.TrimSpace(in.Name)
	v.AllowedCIDRs = in.AllowedCIDRs
	v.CustomSensitiveFields = in.CustomSensitiveFields
	v.MatchMode = in.MatchMode
	v.PayloadPolicy = in.PayloadPolicy
	v.Enabled = enabled
	if in.HMACEnabled && len(v.HMACSecretEnc) == 0 {
		hmacSecret, _ = secure.RandomToken(32)
		v.HMACSecretEnc, err = s.vault.Encrypt([]byte(hmacSecret))
		if err != nil {
			dbError(w, r, err)
			return
		}
	} else if !in.HMACEnabled {
		v.HMACSecretEnc = nil
	}
	v, err = s.store.SaveSource(r.Context(), v)
	if err != nil {
		dbError(w, r, err)
		return
	}
	out := map[string]any{"source": v, "hmac_enabled": len(v.HMACSecretEnc) > 0}
	if token != "" {
		out["token"] = token
		out["hook_url"] = requestBaseURL(r) + "/hooks/" + token
	}
	if hmacSecret != "" {
		out["hmac_secret"] = hmacSecret
	}
	status := 200
	if id == "" {
		status = 201
	}
	writeJSON(w, status, out)
}
func (s *Server) deleteSource(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteSource(r.Context(), pathID(r)); err != nil {
		dbError(w, r, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) rotateSource(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListSources(r.Context())
	if err != nil {
		dbError(w, r, err)
		return
	}
	var v store.Source
	for _, item := range items {
		if item.ID == pathID(r) {
			v = item
		}
	}
	if v.ID == "" {
		writeError(w, 404, "not_found", "source not found", nil, r)
		return
	}
	token, _ := secure.RandomToken(32)
	v.TokenHash = secure.HashToken(token)
	v.TokenPrefix = token[:8]
	if _, err = s.store.SaveSource(r.Context(), v); err != nil {
		dbError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"token": token, "hook_url": requestBaseURL(r) + "/hooks/" + token})
}

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	v, err := s.store.ListRules(r.Context())
	if err != nil {
		dbError(w, r, err)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) saveRule(w http.ResponseWriter, r *http.Request) {
	var v store.Rule
	if !decodeJSON(w, r, &v) {
		return
	}
	v.ID = pathID(r)
	if err := requireName(v.Name); err != nil {
		writeError(w, 400, "validation_error", err.Error(), nil, r)
		return
	}
	if _, err := rules.Parse(v.Condition); err != nil {
		writeError(w, 400, "condition_error", err.Error(), nil, r)
		return
	}
	saved, err := s.store.SaveRule(r.Context(), v)
	if err != nil {
		dbError(w, r, err)
		return
	}
	status := 200
	if pathID(r) == "" {
		status = 201
	}
	writeJSON(w, status, saved)
}
func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteRule(r.Context(), pathID(r)); err != nil {
		dbError(w, r, err)
		return
	}
	w.WriteHeader(204)
}

func (s *Server) listSchedules(w http.ResponseWriter, r *http.Request) {
	v, err := s.store.ListSchedules(r.Context())
	if err != nil {
		dbError(w, r, err)
		return
	}
	for i := range v {
		if raw, e := s.vault.Decrypt(v[i].PayloadEnc); e == nil {
			v[i].Payload = json.RawMessage(raw)
		}
	}
	writeJSON(w, 200, v)
}
func (s *Server) saveSchedule(w http.ResponseWriter, r *http.Request) {
	var in struct {
		store.Schedule
		Payload json.RawMessage `json:"payload"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	v := in.Schedule
	v.ID = pathID(r)
	if err := requireName(v.Name); err != nil {
		writeError(w, 400, "validation_error", err.Error(), nil, r)
		return
	}
	if v.Timezone == "" {
		v.Timezone = s.currentSettings().DefaultTimezone
	}
	var recurrence map[string]any
	if json.Unmarshal(v.Recurrence, &recurrence) == nil {
		kind, _ := recurrence["kind"].(string)
		if kind != "once" && kind != "cron" && recurrence["start_at"] == nil {
			loc, _ := time.LoadLocation(v.Timezone)
			recurrence["start_at"] = time.Now().In(loc).Format(time.RFC3339)
			v.Recurrence, _ = json.Marshal(recurrence)
		}
	}
	next, err := schedule.Next(v.Recurrence, v.Timezone, time.Now().Add(-time.Minute))
	if err != nil {
		writeError(w, 400, "schedule_error", err.Error(), nil, r)
		return
	}
	if v.Enabled && next == nil {
		writeError(w, 400, "schedule_error", "schedule has no future occurrence", nil, r)
		return
	}
	if next != nil {
		unix := next.Unix()
		v.NextRunAt = &unix
	}
	v.PayloadEnc, err = s.vault.Encrypt(in.Payload)
	if err != nil {
		dbError(w, r, err)
		return
	}
	saved, err := s.store.SaveSchedule(r.Context(), v)
	if err != nil {
		dbError(w, r, err)
		return
	}
	saved.Payload = in.Payload
	status := 200
	if pathID(r) == "" {
		status = 201
	}
	writeJSON(w, status, saved)
}
func (s *Server) deleteSchedule(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteSchedule(r.Context(), pathID(r)); err != nil {
		dbError(w, r, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) previewSchedule(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Recurrence json.RawMessage `json:"recurrence"`
		Timezone   string          `json:"timezone"`
		Count      int             `json:"count"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Timezone == "" {
		in.Timezone = s.currentSettings().DefaultTimezone
	}
	if in.Count < 1 || in.Count > 20 {
		in.Count = 5
	}
	after := time.Now()
	out := []int64{}
	for range in.Count {
		next, err := schedule.Next(in.Recurrence, in.Timezone, after)
		if err != nil {
			writeError(w, 400, "schedule_error", err.Error(), nil, r)
			return
		}
		if next == nil {
			break
		}
		out = append(out, next.Unix())
		after = *next
	}
	writeJSON(w, 200, map[string]any{"occurrences": out})
}

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	v, err := s.store.ListEvents(r.Context(), limit)
	if err != nil {
		dbError(w, r, err)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) listDeliveries(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	v, err := s.store.ListDeliveries(r.Context(), limit)
	if err != nil {
		dbError(w, r, err)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) retryDelivery(w http.ResponseWriter, r *http.Request) {
	if err := s.store.RetryJob(r.Context(), pathID(r)); err != nil {
		dbError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"status": "pending"})
}

func (s *Server) testChannel(w http.ResponseWriter, r *http.Request) {
	channel, err := s.store.Channel(r.Context(), pathID(r))
	if err != nil {
		dbError(w, r, err)
		return
	}
	var in struct {
		Body json.RawMessage `json:"body"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	raw, err := s.vault.Decrypt(channel.ConfigEnc)
	if err != nil {
		dbError(w, r, err)
		return
	}
	var cfg map[string]any
	_ = json.Unmarshal(raw, &cfg)
	ctx, cancel := contextWithTimeout(r, 15*time.Second)
	defer cancel()
	status, excerpt, err := s.channels.Send(ctx, channel.Type, cfg, in.Body)
	if err != nil {
		writeError(w, 502, "delivery_failed", err.Error(), map[string]any{"http_status": status, "response": excerpt}, r)
		return
	}
	writeJSON(w, 200, map[string]any{"status": "success", "http_status": status, "response": excerpt})
}
func contextWithTimeout(r *http.Request, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), d)
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.currentSettings())
}
func (s *Server) saveSettings(w http.ResponseWriter, r *http.Request) {
	var input struct {
		DefaultTimezone            string `json:"default_timezone"`
		PayloadRetentionDays       int    `json:"payload_retention_days"`
		MetadataRetentionDays      int    `json:"metadata_retention_days"`
		AllowPrivateWebhookTargets bool   `json:"allow_private_webhook_targets"`
		PocketIDEnabled            bool   `json:"pocketid_enabled"`
		PocketIDIssuerURL          string `json:"pocketid_issuer_url"`
		PocketIDClientID           string `json:"pocketid_client_id"`
		PocketIDClientSecret       string `json:"pocketid_client_secret"`
		PocketIDClearClientSecret  bool   `json:"pocketid_clear_client_secret"`
		PocketIDAllowedIdentity    string `json:"pocketid_allowed_identity"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	current := s.currentSettings()
	settings := store.RuntimeSettings{
		DefaultTimezone:            input.DefaultTimezone,
		PayloadRetentionDays:       input.PayloadRetentionDays,
		MetadataRetentionDays:      input.MetadataRetentionDays,
		AllowPrivateWebhookTargets: input.AllowPrivateWebhookTargets,
		PocketIDEnabled:            input.PocketIDEnabled,
		PocketIDIssuerURL:          normalizePocketIDIssuer(input.PocketIDIssuerURL),
		PocketIDClientID:           strings.TrimSpace(input.PocketIDClientID),
		PocketIDClientSecretEnc:    current.PocketIDClientSecretEnc,
		PocketIDClientSecretSet:    current.PocketIDClientSecretSet,
		PocketIDAllowedIdentity:    strings.TrimSpace(input.PocketIDAllowedIdentity),
	}
	settings.DefaultTimezone = strings.TrimSpace(settings.DefaultTimezone)
	if _, err := time.LoadLocation(settings.DefaultTimezone); err != nil {
		writeError(w, 400, "invalid_timezone", "default_timezone must be a valid IANA timezone", nil, r)
		return
	}
	if settings.PayloadRetentionDays < 0 || settings.PayloadRetentionDays > 3650 {
		writeError(w, 400, "invalid_retention", "payload_retention_days must be between 0 and 3650", nil, r)
		return
	}
	if settings.MetadataRetentionDays < 1 || settings.MetadataRetentionDays > 3650 {
		writeError(w, 400, "invalid_retention", "metadata_retention_days must be between 1 and 3650", nil, r)
		return
	}
	if settings.PayloadRetentionDays > settings.MetadataRetentionDays {
		writeError(w, 400, "invalid_retention", "payload_retention_days cannot exceed metadata_retention_days", nil, r)
		return
	}
	if input.PocketIDClearClientSecret {
		settings.PocketIDClientSecretEnc = ""
		settings.PocketIDClientSecretSet = false
	}
	if secret := strings.TrimSpace(input.PocketIDClientSecret); secret != "" {
		encrypted, err := s.vault.Encrypt([]byte(secret))
		if err != nil {
			writeError(w, 500, "encryption_failed", err.Error(), nil, r)
			return
		}
		settings.PocketIDClientSecretEnc = base64.RawStdEncoding.EncodeToString(encrypted)
		settings.PocketIDClientSecretSet = true
	}
	if settings.PocketIDIssuerURL != "" {
		if err := validatePocketIDIssuer(settings.PocketIDIssuerURL); err != nil {
			writeError(w, 400, "invalid_pocketid_issuer", err.Error(), nil, r)
			return
		}
	}
	if settings.PocketIDEnabled && !pocketIDConfigured(settings) {
		writeError(w, 400, "incomplete_pocketid_settings", "Pocket ID issuer URL, client ID, client secret, and allowed identity are required when enabled", nil, r)
		return
	}
	if err := s.store.SaveRuntimeSettings(r.Context(), settings); err != nil {
		dbError(w, r, err)
		return
	}
	s.settings.Store(settings)
	s.channels.SetAllowPrivate(settings.AllowPrivateWebhookTargets)
	writeJSON(w, 200, settings)
}
func sampleContext() map[string]any {
	return map[string]any{"body": map[string]any{"title": "示例告警", "message": "服务恢复正常", "status": "ok"}, "raw_body": "{\"status\":\"ok\"}", "headers": map[string]any{"content-type": "application/json"}, "query": map[string]any{}, "form": map[string]any{}, "meta": map[string]any{"event_id": "preview", "received_at": time.Now().UTC().Format(time.RFC3339), "method": "POST"}}
}

var _ = fmt.Sprint
