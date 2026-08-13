package channels

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var Supported = map[string]bool{"dingtalk": true, "wecom": true, "wecom_app": true, "feishu": true, "telegram": true, "discord": true, "bark": true, "webhook": true}

type Service struct {
	base         *http.Client
	allowPrivate atomic.Bool
	telegramBase string
	wecomBase    string
}

func New(allowPrivate bool) *Service {
	service := &Service{base: &http.Client{Timeout: 15 * time.Second}, telegramBase: "https://api.telegram.org", wecomBase: "https://qyapi.weixin.qq.com"}
	service.allowPrivate.Store(allowPrivate)
	return service
}

func (s *Service) SetAllowPrivate(allow bool) { s.allowPrivate.Store(allow) }

func Validate(channelType string, config map[string]any) error {
	if !Supported[channelType] {
		return fmt.Errorf("unsupported channel type %q", channelType)
	}
	required := map[string][]string{"dingtalk": {"webhook"}, "wecom": {"webhook"}, "wecom_app": {"corp_id", "agent_id", "secret"}, "feishu": {"webhook"}, "telegram": {"bot_token", "chat_id"}, "discord": {"webhook"}, "bark": {"webhook"}, "webhook": {"webhook"}}
	for _, key := range required[channelType] {
		if strings.TrimSpace(fmt.Sprint(config[key])) == "" {
			return fmt.Errorf("%s is required", key)
		}
	}
	return nil
}

func (s *Service) Send(ctx context.Context, typ string, config map[string]any, body json.RawMessage) (int, string, error) {
	if err := Validate(typ, config); err != nil {
		return 0, "", err
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, "", err
	}
	switch typ {
	case "telegram":
		return s.telegram(ctx, config, payload)
	case "wecom_app":
		return s.wecomApp(ctx, config, payload)
	case "dingtalk":
		return s.dingtalk(ctx, config, payload)
	case "feishu":
		return s.feishu(ctx, config, payload)
	case "webhook":
		return s.webhook(ctx, config, payload)
	case "wecom":
		status, excerpt, err := s.postJSON(ctx, s.base, fmt.Sprint(config["webhook"]), payload)
		return platformResult(status, excerpt, err, "errcode", "errmsg")
	default:
		return s.postJSON(ctx, s.base, fmt.Sprint(config["webhook"]), payload)
	}
}

func (s *Service) postJSON(ctx context.Context, client *http.Client, target string, payload any) (int, string, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return 0, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(b))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	return do(client, req)
}
func do(client *http.Client, req *http.Request) (int, string, error) {
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	excerpt := strings.TrimSpace(string(b))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, excerpt, fmt.Errorf("remote returned HTTP %d", resp.StatusCode)
	}
	return resp.StatusCode, excerpt, nil
}

func (s *Service) telegram(ctx context.Context, c map[string]any, p any) (int, string, error) {
	m, ok := p.(map[string]any)
	if !ok {
		return 0, "", errors.New("telegram payload must be an object")
	}
	m["chat_id"] = fmt.Sprint(c["chat_id"])
	target := s.telegramBase + "/bot" + url.PathEscape(fmt.Sprint(c["bot_token"])) + "/sendMessage"
	return s.postJSON(ctx, s.base, target, m)
}

func (s *Service) wecomApp(ctx context.Context, c map[string]any, p any) (int, string, error) {
	target := s.wecomBase + "/cgi-bin/gettoken?corpid=" + url.QueryEscape(fmt.Sprint(c["corp_id"])) + "&corpsecret=" + url.QueryEscape(fmt.Sprint(c["secret"]))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	resp, err := s.base.Do(req)
	if err != nil {
		return 0, "", err
	}
	var token struct {
		AccessToken string `json:"access_token"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&token)
	resp.Body.Close()
	if err != nil || token.AccessToken == "" {
		return resp.StatusCode, "", fmt.Errorf("get WeCom access token: %s", token.ErrMsg)
	}
	m, ok := p.(map[string]any)
	if !ok {
		return 0, "", errors.New("WeCom app payload must be an object")
	}
	agentID, err := strconv.Atoi(fmt.Sprint(c["agent_id"]))
	if err != nil {
		return 0, "", errors.New("agent_id must be an integer")
	}
	m["agentid"] = agentID
	if _, ok = m["touser"]; !ok {
		m["touser"] = "@all"
	}
	status, excerpt, err := s.postJSON(ctx, s.base, s.wecomBase+"/cgi-bin/message/send?access_token="+url.QueryEscape(token.AccessToken), m)
	return platformResult(status, excerpt, err, "errcode", "errmsg")
}

func (s *Service) dingtalk(ctx context.Context, c map[string]any, p any) (int, string, error) {
	target := fmt.Sprint(c["webhook"])
	if secret := strings.TrimSpace(fmt.Sprint(c["secret"])); secret != "" {
		ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(ts + "\n" + secret))
		u, err := url.Parse(target)
		if err != nil {
			return 0, "", err
		}
		q := u.Query()
		q.Set("timestamp", ts)
		q.Set("sign", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		u.RawQuery = q.Encode()
		target = u.String()
	}
	status, excerpt, err := s.postJSON(ctx, s.base, target, p)
	return platformResult(status, excerpt, err, "errcode", "errmsg")
}

func (s *Service) feishu(ctx context.Context, c map[string]any, p any) (int, string, error) {
	m, ok := p.(map[string]any)
	if !ok {
		return 0, "", errors.New("Feishu payload must be an object")
	}
	if secret := strings.TrimSpace(fmt.Sprint(c["secret"])); secret != "" {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		mac := hmac.New(sha256.New, []byte(ts+"\n"+secret))
		m["timestamp"] = ts
		m["sign"] = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	}
	status, excerpt, err := s.postJSON(ctx, s.base, fmt.Sprint(c["webhook"]), m)
	return platformResult(status, excerpt, err, "code", "msg")
}

func platformResult(status int, excerpt string, requestErr error, codeKey, messageKey string) (int, string, error) {
	if requestErr != nil {
		return status, excerpt, requestErr
	}
	if excerpt == "" {
		return status, excerpt, nil
	}
	var response map[string]any
	if json.Unmarshal([]byte(excerpt), &response) != nil {
		return status, excerpt, nil
	}
	code, exists := response[codeKey]
	if !exists {
		return status, excerpt, nil
	}
	if number, ok := code.(float64); ok && number == 0 {
		return status, excerpt, nil
	}
	if text := fmt.Sprint(code); text == "0" || strings.EqualFold(text, "success") {
		return status, excerpt, nil
	}
	return status, excerpt, fmt.Errorf("platform returned %s: %v", codeKey, response[messageKey])
}

func (s *Service) webhook(ctx context.Context, c map[string]any, p any) (int, string, error) {
	m, ok := p.(map[string]any)
	if !ok {
		return 0, "", errors.New("webhook payload must be an object")
	}
	target := fmt.Sprint(c["webhook"])
	u, err := url.Parse(target)
	if err != nil {
		return 0, "", err
	}
	if q, ok := m["query"].(map[string]any); ok {
		values := u.Query()
		for k, v := range q {
			values.Set(k, fmt.Sprint(v))
		}
		u.RawQuery = values.Encode()
	}
	client, err := s.safeClient(u)
	if err != nil {
		return 0, "", err
	}
	method := strings.ToUpper(fmt.Sprint(m["method"]))
	if method == "" || method == "<NIL>" {
		method = http.MethodPost
	}
	if method != "GET" && method != "POST" && method != "PUT" && method != "PATCH" {
		return 0, "", errors.New("unsupported outbound webhook method")
	}
	var reader io.Reader
	if method != "GET" {
		switch body := m["body"].(type) {
		case string:
			reader = strings.NewReader(body)
		default:
			b, _ := json.Marshal(body)
			reader = bytes.NewReader(b)
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return 0, "", err
	}
	if headers, ok := m["headers"].(map[string]any); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprint(v))
		}
	}
	if req.Header.Get("Content-Type") == "" && method != "GET" {
		req.Header.Set("Content-Type", "application/json")
	}
	return do(client, req)
}

func (s *Service) safeClient(u *url.URL) (*http.Client, error) {
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, errors.New("webhook URL must use http or https")
	}
	if err := s.validateHost(u.Hostname()); err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, errors.New("webhook host did not resolve")
		}
		for _, ip := range ips {
			if err = s.validateIP(ip); err != nil {
				return nil, err
			}
		}
		if err = s.validateHost(host); err != nil {
			return nil, err
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}}
	client := &http.Client{Timeout: 15 * time.Second, Transport: transport}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return s.validateHost(req.URL.Hostname())
	}
	return client, nil
}
func (s *Service) validateHost(host string) error {
	if s.allowPrivate.Load() {
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if err := s.validateIP(ip); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) validateIP(ip net.IP) error {
	if s.allowPrivate.Load() {
		return nil
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.String() == "169.254.169.254" {
		return errors.New("private or local webhook targets are blocked")
	}
	return nil
}
