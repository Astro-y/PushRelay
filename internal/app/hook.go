package app

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"pushrelay/internal/rules"
	"pushrelay/internal/secure"
	"pushrelay/internal/store"
)

func (s *Server) hook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" && r.Method != "PUT" && r.Method != "PATCH" {
		w.Header().Set("Allow", "GET, POST, PUT, PATCH")
		writeError(w, 405, "method_not_allowed", "method not allowed", nil, r)
		return
	}
	token := chi.URLParam(r, "token")
	source, err := s.store.SourceByTokenHash(r.Context(), secure.HashToken(token))
	if err != nil || !source.Enabled {
		writeError(w, 404, "hook_not_found", "webhook source not found", nil, r)
		return
	}
	if !allowedIP(r.RemoteAddr, source.AllowedCIDRs) {
		writeError(w, 403, "ip_rejected", "source IP is not allowed", nil, r)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes))
	if err != nil {
		writeError(w, 413, "body_too_large", "request body exceeds the configured limit", nil, r)
		return
	}
	if len(source.HMACSecretEnc) > 0 {
		secret, decErr := s.vault.Decrypt(source.HMACSecretEnc)
		if decErr != nil {
			writeError(w, 500, "secret_error", "cannot decrypt HMAC secret", nil, r)
			return
		}
		timestamp := r.Header.Get("X-Push-Timestamp")
		unix, parseErr := strconv.ParseInt(timestamp, 10, 64)
		if parseErr != nil || time.Since(time.Unix(unix, 0)).Abs() > 5*time.Minute || !secure.VerifyHMAC(string(secret), timestamp, body, r.Header.Get("X-Push-Signature")) {
			writeError(w, 401, "invalid_signature", "invalid or expired webhook signature", nil, r)
			return
		}
	}
	envelope, contentType, err := requestEnvelope(r, body)
	if err != nil {
		writeError(w, 400, "invalid_payload", err.Error(), nil, r)
		return
	}
	eventID := store.NewID()
	envelope["meta"] = map[string]any{"event_id": eventID, "received_at": time.Now().UTC().Format(time.RFC3339), "method": r.Method, "source_id": source.ID, "source_name": source.Name}
	ruleItems, err := s.store.RulesForSource(r.Context(), source.ID)
	if err != nil {
		dbError(w, r, err)
		return
	}
	groups := []string{}
	matched := 0
	for _, item := range ruleItems {
		if !item.Enabled {
			continue
		}
		node, parseErr := rules.Parse(item.Condition)
		if parseErr != nil {
			s.logger.Error("stored rule is invalid", "rule_id", item.ID, "error", parseErr)
			continue
		}
		ok, matchErr := rules.Match(node, envelope)
		if matchErr != nil {
			s.logger.Error("rule evaluation failed", "rule_id", item.ID, "error", matchErr)
			continue
		}
		if ok {
			matched++
			groups = append(groups, item.TargetGroupID)
			if source.MatchMode == "first_match" {
				break
			}
		}
	}
	raw, _ := json.Marshal(envelope)
	encrypted, err := s.vault.Encrypt(raw)
	if err != nil {
		dbError(w, r, err)
		return
	}
	id, duplicate, err := s.store.AcceptEvent(r.Context(), store.AcceptEventInput{ID: eventID, SourceID: source.ID, TriggerType: "webhook", IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")), Method: r.Method, ContentType: contentType, PayloadPolicy: source.PayloadPolicy, PayloadEnc: encrypted, MatchedRules: matched, GroupIDs: groups})
	if err != nil {
		dbError(w, r, err)
		return
	}
	if !duplicate && len(groups) == 0 && (source.PayloadPolicy == "none" || source.PayloadPolicy == "metadata") {
		_ = s.store.ReplaceEventPayload(r.Context(), id, nil)
	} else if !duplicate && len(groups) == 0 && source.PayloadPolicy == "redacted" {
		redacted, _ := json.Marshal(rules.RedactWithFields(envelope, source.CustomSensitiveFields))
		if redactedEnc, encryptErr := s.vault.Encrypt(redacted); encryptErr == nil {
			_ = s.store.ReplaceEventPayload(r.Context(), id, redactedEnc)
		}
	}
	writeJSON(w, 202, map[string]any{"event_id": id, "status": "accepted", "duplicate": duplicate, "matched_rules": matched})
}

func requestEnvelope(r *http.Request, raw []byte) (map[string]any, string, error) {
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	out := map[string]any{"raw_body": string(raw), "headers": headerMap(r.Header), "query": valuesMap(r.URL.Query()), "form": map[string]any{}}
	switch mediaType {
	case "application/json", "application/cloudevents+json", "":
		if len(raw) == 0 {
			out["body"] = map[string]any{}
		} else {
			var body any
			dec := json.NewDecoder(strings.NewReader(string(raw)))
			dec.UseNumber()
			if err := dec.Decode(&body); err != nil {
				return nil, mediaType, err
			}
			out["body"] = body
		}
	case "application/x-www-form-urlencoded":
		values, err := url.ParseQuery(string(raw))
		if err != nil {
			return nil, mediaType, err
		}
		form := valuesMap(values)
		out["form"] = form
		out["body"] = form
	case "text/plain":
		out["body"] = string(raw)
	default:
		return nil, mediaType, errors.New("supported content types are application/json, application/x-www-form-urlencoded and text/plain")
	}
	return out, mediaType, nil
}
func headerMap(h http.Header) map[string]any {
	out := map[string]any{}
	for k, v := range h {
		key := strings.ToLower(k)
		if len(v) == 1 {
			out[key] = v[0]
		} else {
			out[key] = v
		}
	}
	return out
}
func valuesMap(values url.Values) map[string]any {
	out := map[string]any{}
	for k, v := range values {
		if len(v) == 1 {
			out[k] = v[0]
		} else {
			out[k] = v
		}
	}
	return out
}
func allowedIP(remote string, cidrs []string) bool {
	if len(cidrs) == 0 {
		return true
	}
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		host = remote
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, raw := range cidrs {
		if one := net.ParseIP(raw); one != nil && one.Equal(ip) {
			return true
		}
		_, network, e := net.ParseCIDR(raw)
		if e == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}
