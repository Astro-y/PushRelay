package store

import "encoding/json"

type Channel struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	ConfigEnc []byte         `json:"-"`
	Config    map[string]any `json:"config,omitempty"`
	Enabled   bool           `json:"enabled"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
}

type MessageTemplate struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	ChannelType string          `json:"channel_type"`
	Body        json.RawMessage `json:"body"`
	CreatedAt   int64           `json:"created_at"`
	UpdatedAt   int64           `json:"updated_at"`
}

type TargetBinding struct {
	ID           string `json:"id"`
	GroupID      string `json:"group_id"`
	ChannelID    string `json:"channel_id"`
	TemplateID   string `json:"template_id"`
	ChannelName  string `json:"channel_name,omitempty"`
	TemplateName string `json:"template_name,omitempty"`
	Enabled      bool   `json:"enabled"`
}

type TargetGroup struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Bindings  []TargetBinding `json:"bindings"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
}

type Source struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	TokenHash             string   `json:"-"`
	TokenPrefix           string   `json:"token_prefix"`
	HMACSecretEnc         []byte   `json:"-"`
	AllowedCIDRs          []string `json:"allowed_cidrs"`
	CustomSensitiveFields []string `json:"custom_sensitive_fields"`
	MatchMode             string   `json:"match_mode"`
	PayloadPolicy         string   `json:"payload_policy"`
	Enabled               bool     `json:"enabled"`
	CreatedAt             int64    `json:"created_at"`
	UpdatedAt             int64    `json:"updated_at"`
}

type Rule struct {
	ID            string          `json:"id"`
	SourceID      string          `json:"source_id"`
	Name          string          `json:"name"`
	Priority      int             `json:"priority"`
	Condition     json.RawMessage `json:"condition"`
	TargetGroupID string          `json:"target_group_id"`
	Enabled       bool            `json:"enabled"`
	CreatedAt     int64           `json:"created_at"`
	UpdatedAt     int64           `json:"updated_at"`
}

type Schedule struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Recurrence    json.RawMessage `json:"recurrence"`
	Timezone      string          `json:"timezone"`
	PayloadEnc    []byte          `json:"-"`
	Payload       json.RawMessage `json:"payload,omitempty"`
	TargetGroupID string          `json:"target_group_id"`
	Enabled       bool            `json:"enabled"`
	NextRunAt     *int64          `json:"next_run_at"`
	LastRunAt     *int64          `json:"last_run_at"`
	CreatedAt     int64           `json:"created_at"`
	UpdatedAt     int64           `json:"updated_at"`
}

type Event struct {
	ID            string  `json:"id"`
	SourceID      *string `json:"source_id"`
	ScheduleID    *string `json:"schedule_id"`
	TriggerType   string  `json:"trigger_type"`
	Method        string  `json:"method"`
	ContentType   string  `json:"content_type"`
	PayloadPolicy string  `json:"payload_policy"`
	MatchedRules  int     `json:"matched_rules"`
	CreatedAt     int64   `json:"created_at"`
}

type Delivery struct {
	ID           string  `json:"id"`
	EventID      string  `json:"event_id"`
	Status       string  `json:"status"`
	Attempts     int     `json:"attempts"`
	RunAfter     int64   `json:"run_after"`
	LastError    *string `json:"last_error"`
	ChannelName  string  `json:"channel_name"`
	TemplateName string  `json:"template_name"`
	CreatedAt    int64   `json:"created_at"`
	UpdatedAt    int64   `json:"updated_at"`
}

type JobPayload struct {
	JobID      string
	EventID    string
	BindingID  string
	Attempts   int
	PayloadEnc []byte
	Channel    Channel
	Template   MessageTemplate
}
