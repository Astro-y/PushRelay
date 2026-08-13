package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (s *Store) ListChannels(ctx context.Context) ([]Channel, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,name,type,config_enc,enabled,created_at,updated_at FROM channels ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Channel
	for rows.Next() {
		var v Channel
		if err = rows.Scan(&v.ID, &v.Name, &v.Type, &v.ConfigEnc, &v.Enabled, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) Channel(ctx context.Context, id string) (Channel, error) {
	var v Channel
	err := s.DB.QueryRowContext(ctx, `SELECT id,name,type,config_enc,enabled,created_at,updated_at FROM channels WHERE id=?`, id).Scan(&v.ID, &v.Name, &v.Type, &v.ConfigEnc, &v.Enabled, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}

func (s *Store) SaveChannel(ctx context.Context, v Channel) (Channel, error) {
	now := NowUnix()
	if v.ID == "" {
		v.ID = NewID()
		v.CreatedAt = now
	}
	v.UpdatedAt = now
	_, err := s.DB.ExecContext(ctx, `INSERT INTO channels(id,name,type,config_enc,enabled,created_at,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,type=excluded.type,config_enc=excluded.config_enc,enabled=excluded.enabled,updated_at=excluded.updated_at`, v.ID, v.Name, v.Type, v.ConfigEnc, boolInt(v.Enabled), v.CreatedAt, v.UpdatedAt)
	return v, err
}

func (s *Store) DeleteChannel(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM channels WHERE id=?`, id)
	return err
}

func (s *Store) ListTemplates(ctx context.Context) ([]MessageTemplate, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,name,channel_type,body_json,created_at,updated_at FROM message_templates ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MessageTemplate
	for rows.Next() {
		var v MessageTemplate
		var body string
		if err = rows.Scan(&v.ID, &v.Name, &v.ChannelType, &body, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		v.Body = json.RawMessage(body)
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) SaveTemplate(ctx context.Context, v MessageTemplate) (MessageTemplate, error) {
	now := NowUnix()
	if v.ID == "" {
		v.ID = NewID()
		v.CreatedAt = now
	}
	v.UpdatedAt = now
	_, err := s.DB.ExecContext(ctx, `INSERT INTO message_templates(id,name,channel_type,body_json,created_at,updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,channel_type=excluded.channel_type,body_json=excluded.body_json,updated_at=excluded.updated_at`, v.ID, v.Name, v.ChannelType, string(v.Body), v.CreatedAt, v.UpdatedAt)
	return v, err
}
func (s *Store) DeleteTemplate(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM message_templates WHERE id=?`, id)
	return err
}

func (s *Store) ListGroups(ctx context.Context) ([]TargetGroup, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,name,created_at,updated_at FROM target_groups ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TargetGroup
	for rows.Next() {
		var g TargetGroup
		if err = rows.Scan(&g.ID, &g.Name, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		g.Bindings, err = s.GroupBindings(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
func (s *Store) GroupBindings(ctx context.Context, id string) ([]TargetBinding, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT b.id,b.group_id,b.channel_id,b.template_id,c.name,t.name,b.enabled FROM target_bindings b JOIN channels c ON c.id=b.channel_id JOIN message_templates t ON t.id=b.template_id WHERE b.group_id=? ORDER BY c.name`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TargetBinding
	for rows.Next() {
		var v TargetBinding
		if err = rows.Scan(&v.ID, &v.GroupID, &v.ChannelID, &v.TemplateID, &v.ChannelName, &v.TemplateName, &v.Enabled); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) SaveGroup(ctx context.Context, g TargetGroup) (TargetGroup, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return g, err
	}
	defer tx.Rollback()
	now := NowUnix()
	if g.ID == "" {
		g.ID = NewID()
		g.CreatedAt = now
	}
	g.UpdatedAt = now
	if _, err = tx.ExecContext(ctx, `INSERT INTO target_groups(id,name,created_at,updated_at) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,updated_at=excluded.updated_at`, g.ID, g.Name, g.CreatedAt, g.UpdatedAt); err != nil {
		return g, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM target_bindings WHERE group_id=?`, g.ID); err != nil {
		return g, err
	}
	for i := range g.Bindings {
		b := &g.Bindings[i]
		if b.ID == "" {
			b.ID = NewID()
		}
		b.GroupID = g.ID
		if _, err = tx.ExecContext(ctx, `INSERT INTO target_bindings(id,group_id,channel_id,template_id,enabled) VALUES(?,?,?,?,?)`, b.ID, g.ID, b.ChannelID, b.TemplateID, boolInt(b.Enabled)); err != nil {
			return g, err
		}
	}
	return g, tx.Commit()
}
func (s *Store) DeleteGroup(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM target_groups WHERE id=?`, id)
	return err
}

func scanSource(scanner interface{ Scan(...any) error }) (Source, error) {
	var v Source
	var cidrs, sensitive string
	err := scanner.Scan(&v.ID, &v.Name, &v.TokenHash, &v.TokenPrefix, &v.HMACSecretEnc, &cidrs, &v.MatchMode, &sensitive, &v.PayloadPolicy, &v.Enabled, &v.CreatedAt, &v.UpdatedAt)
	if err == nil {
		_ = json.Unmarshal([]byte(cidrs), &v.AllowedCIDRs)
		_ = json.Unmarshal([]byte(sensitive), &v.CustomSensitiveFields)
	}
	return v, err
}
func (s *Store) ListSources(ctx context.Context) ([]Source, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,name,token_hash,token_prefix,hmac_secret_enc,allowed_cidrs,match_mode,custom_sensitive_fields,payload_policy,enabled,created_at,updated_at FROM webhook_sources ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Source
	for rows.Next() {
		v, e := scanSource(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) SourceByTokenHash(ctx context.Context, h string) (Source, error) {
	return scanSource(s.DB.QueryRowContext(ctx, `SELECT id,name,token_hash,token_prefix,hmac_secret_enc,allowed_cidrs,match_mode,custom_sensitive_fields,payload_policy,enabled,created_at,updated_at FROM webhook_sources WHERE token_hash=?`, h))
}
func (s *Store) SaveSource(ctx context.Context, v Source) (Source, error) {
	now := NowUnix()
	if v.ID == "" {
		v.ID = NewID()
		v.CreatedAt = now
	}
	v.UpdatedAt = now
	cidrs, _ := json.Marshal(v.AllowedCIDRs)
	sensitive, _ := json.Marshal(v.CustomSensitiveFields)
	_, err := s.DB.ExecContext(ctx, `INSERT INTO webhook_sources(id,name,token_hash,token_prefix,hmac_secret_enc,allowed_cidrs,match_mode,custom_sensitive_fields,payload_policy,enabled,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,token_hash=excluded.token_hash,token_prefix=excluded.token_prefix,hmac_secret_enc=excluded.hmac_secret_enc,allowed_cidrs=excluded.allowed_cidrs,match_mode=excluded.match_mode,custom_sensitive_fields=excluded.custom_sensitive_fields,payload_policy=excluded.payload_policy,enabled=excluded.enabled,updated_at=excluded.updated_at`, v.ID, v.Name, v.TokenHash, v.TokenPrefix, v.HMACSecretEnc, string(cidrs), v.MatchMode, string(sensitive), v.PayloadPolicy, boolInt(v.Enabled), v.CreatedAt, v.UpdatedAt)
	return v, err
}
func (s *Store) DeleteSource(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM webhook_sources WHERE id=?`, id)
	return err
}

func (s *Store) RulesForSource(ctx context.Context, sourceID string) ([]Rule, error) {
	return s.listRules(ctx, `WHERE source_id=?`, sourceID)
}
func (s *Store) ListRules(ctx context.Context) ([]Rule, error) { return s.listRules(ctx, "", nil) }
func (s *Store) listRules(ctx context.Context, where string, arg any) ([]Rule, error) {
	q := `SELECT id,source_id,name,priority,condition_json,target_group_id,enabled,created_at,updated_at FROM rules ` + where + ` ORDER BY priority ASC,created_at ASC`
	var rows *sql.Rows
	var err error
	if where == "" {
		rows, err = s.DB.QueryContext(ctx, q)
	} else {
		rows, err = s.DB.QueryContext(ctx, q, arg)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Rule
	for rows.Next() {
		var v Rule
		var cond string
		if err = rows.Scan(&v.ID, &v.SourceID, &v.Name, &v.Priority, &cond, &v.TargetGroupID, &v.Enabled, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		v.Condition = json.RawMessage(cond)
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) SaveRule(ctx context.Context, v Rule) (Rule, error) {
	now := NowUnix()
	if v.ID == "" {
		v.ID = NewID()
		v.CreatedAt = now
	}
	v.UpdatedAt = now
	_, err := s.DB.ExecContext(ctx, `INSERT INTO rules(id,source_id,name,priority,condition_json,target_group_id,enabled,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET source_id=excluded.source_id,name=excluded.name,priority=excluded.priority,condition_json=excluded.condition_json,target_group_id=excluded.target_group_id,enabled=excluded.enabled,updated_at=excluded.updated_at`, v.ID, v.SourceID, v.Name, v.Priority, string(v.Condition), v.TargetGroupID, boolInt(v.Enabled), v.CreatedAt, v.UpdatedAt)
	return v, err
}
func (s *Store) DeleteRule(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM rules WHERE id=?`, id)
	return err
}

func (s *Store) ListSchedules(ctx context.Context) ([]Schedule, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,name,recurrence_json,timezone,payload_enc,target_group_id,enabled,next_run_at,last_run_at,created_at,updated_at FROM schedules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Schedule
	for rows.Next() {
		v, e := scanSchedule(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func scanSchedule(scanner interface{ Scan(...any) error }) (Schedule, error) {
	var v Schedule
	var rec string
	var next, last sql.NullInt64
	err := scanner.Scan(&v.ID, &v.Name, &rec, &v.Timezone, &v.PayloadEnc, &v.TargetGroupID, &v.Enabled, &next, &last, &v.CreatedAt, &v.UpdatedAt)
	v.Recurrence = json.RawMessage(rec)
	if next.Valid {
		v.NextRunAt = &next.Int64
	}
	if last.Valid {
		v.LastRunAt = &last.Int64
	}
	return v, err
}
func (s *Store) SaveSchedule(ctx context.Context, v Schedule) (Schedule, error) {
	now := NowUnix()
	if v.ID == "" {
		v.ID = NewID()
		v.CreatedAt = now
	}
	v.UpdatedAt = now
	_, err := s.DB.ExecContext(ctx, `INSERT INTO schedules(id,name,recurrence_json,timezone,payload_enc,target_group_id,enabled,next_run_at,last_run_at,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,recurrence_json=excluded.recurrence_json,timezone=excluded.timezone,payload_enc=excluded.payload_enc,target_group_id=excluded.target_group_id,enabled=excluded.enabled,next_run_at=excluded.next_run_at,updated_at=excluded.updated_at`, v.ID, v.Name, string(v.Recurrence), v.Timezone, v.PayloadEnc, v.TargetGroupID, boolInt(v.Enabled), v.NextRunAt, v.LastRunAt, v.CreatedAt, v.UpdatedAt)
	return v, err
}
func (s *Store) DeleteSchedule(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM schedules WHERE id=?`, id)
	return err
}
func (s *Store) DueSchedules(ctx context.Context, now int64) ([]Schedule, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,name,recurrence_json,timezone,payload_enc,target_group_id,enabled,next_run_at,last_run_at,created_at,updated_at FROM schedules WHERE enabled=1 AND next_run_at IS NOT NULL AND next_run_at<=? ORDER BY next_run_at LIMIT 32`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Schedule
	for rows.Next() {
		v, e := scanSchedule(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) MarkScheduleRun(ctx context.Context, id string, last int64, next *int64) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE schedules SET last_run_at=?,next_run_at=?,enabled=CASE WHEN ? IS NULL THEN 0 ELSE enabled END,updated_at=? WHERE id=?`, last, next, next, NowUnix(), id)
	return err
}

type AcceptEventInput struct {
	ID, SourceID, ScheduleID, TriggerType, IdempotencyKey, Method, ContentType, PayloadPolicy string
	PayloadEnc                                                                                []byte
	MatchedRules                                                                              int
	GroupIDs                                                                                  []string
}

func (s *Store) AcceptEvent(ctx context.Context, in AcceptEventInput) (string, bool, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback()
	if in.IdempotencyKey != "" && in.SourceID != "" {
		var old string
		err = tx.QueryRowContext(ctx, `SELECT id FROM events WHERE source_id=? AND idempotency_key=? AND created_at>?`, in.SourceID, in.IdempotencyKey, time.Now().Add(-24*time.Hour).Unix()).Scan(&old)
		if err == nil {
			return old, true, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return "", false, err
		}
	}
	if in.ID == "" {
		in.ID = NewID()
	}
	var source, schedule any
	if in.SourceID != "" {
		source = in.SourceID
	}
	if in.ScheduleID != "" {
		schedule = in.ScheduleID
	}
	var idem any
	if in.IdempotencyKey != "" {
		idem = in.IdempotencyKey
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO events(id,source_id,schedule_id,trigger_type,idempotency_key,method,content_type,payload_enc,payload_policy,matched_rules,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, in.ID, source, schedule, in.TriggerType, idem, in.Method, in.ContentType, in.PayloadEnc, in.PayloadPolicy, in.MatchedRules, NowUnix())
	if err != nil {
		return "", false, err
	}
	seen := map[string]bool{}
	for _, gid := range in.GroupIDs {
		rows, e := tx.QueryContext(ctx, `SELECT id FROM target_bindings WHERE group_id=? AND enabled=1`, gid)
		if e != nil {
			return "", false, e
		}
		for rows.Next() {
			var bid string
			if e = rows.Scan(&bid); e != nil {
				rows.Close()
				return "", false, e
			}
			if seen[bid] {
				continue
			}
			seen[bid] = true
			now := NowUnix()
			_, e = tx.ExecContext(ctx, `INSERT OR IGNORE INTO delivery_jobs(id,event_id,binding_id,status,attempts,run_after,created_at,updated_at) VALUES(?,?,?,'pending',0,?,?,?)`, NewID(), in.ID, bid, now, now, now)
			if e != nil {
				rows.Close()
				return "", false, e
			}
		}
		rows.Close()
	}
	return in.ID, false, tx.Commit()
}

func (s *Store) ClaimJob(ctx context.Context) (JobPayload, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return JobPayload{}, err
	}
	defer tx.Rollback()
	var id string
	err = tx.QueryRowContext(ctx, `SELECT id FROM delivery_jobs WHERE (status='pending' OR (status='processing' AND locked_at<?)) AND run_after<=? ORDER BY run_after,created_at LIMIT 1`, time.Now().Add(-5*time.Minute).Unix(), NowUnix()).Scan(&id)
	if err != nil {
		return JobPayload{}, err
	}
	res, err := tx.ExecContext(ctx, `UPDATE delivery_jobs SET status='processing',locked_at=?,updated_at=? WHERE id=? AND status IN ('pending','processing')`, NowUnix(), NowUnix(), id)
	if err != nil {
		return JobPayload{}, err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return JobPayload{}, sql.ErrNoRows
	}
	var p JobPayload
	var body string
	err = tx.QueryRowContext(ctx, `SELECT j.id,j.event_id,j.binding_id,j.attempts,e.payload_enc,c.id,c.name,c.type,c.config_enc,c.enabled,c.created_at,c.updated_at,t.id,t.name,t.channel_type,t.body_json,t.created_at,t.updated_at FROM delivery_jobs j JOIN events e ON e.id=j.event_id JOIN target_bindings b ON b.id=j.binding_id JOIN channels c ON c.id=b.channel_id JOIN message_templates t ON t.id=b.template_id WHERE j.id=?`, id).Scan(&p.JobID, &p.EventID, &p.BindingID, &p.Attempts, &p.PayloadEnc, &p.Channel.ID, &p.Channel.Name, &p.Channel.Type, &p.Channel.ConfigEnc, &p.Channel.Enabled, &p.Channel.CreatedAt, &p.Channel.UpdatedAt, &p.Template.ID, &p.Template.Name, &p.Template.ChannelType, &body, &p.Template.CreatedAt, &p.Template.UpdatedAt)
	p.Template.Body = json.RawMessage(body)
	if err != nil {
		return JobPayload{}, err
	}
	return p, tx.Commit()
}

func (s *Store) FinishAttempt(ctx context.Context, jobID string, attempt int, success bool, httpStatus, duration int, errText, response string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	status := "success"
	jobStatus := "success"
	var runAfter any = NowUnix()
	if !success {
		status = "failed"
		if attempt >= 5 {
			jobStatus = "dead"
		} else {
			jobStatus = "pending"
			delays := []time.Duration{0, 30 * time.Second, 2 * time.Minute, 10 * time.Minute, 30 * time.Minute}
			runAfter = time.Now().Add(delays[attempt-1] + time.Duration(rand.IntN(5000))*time.Millisecond).Unix()
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO delivery_attempts(id,job_id,attempt,status,http_status,duration_ms,error,response_excerpt,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, NewID(), jobID, attempt, status, httpStatus, duration, nullable(errText), nullable(response), NowUnix())
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE delivery_jobs SET status=?,attempts=?,run_after=?,locked_at=NULL,last_error=?,updated_at=? WHERE id=?`, jobStatus, attempt, runAfter, nullable(errText), NowUnix(), jobID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) EventPayloadPolicyIfComplete(ctx context.Context, eventID string) (string, []string, bool, error) {
	var policy, sensitiveJSON string
	var remaining int
	err := s.DB.QueryRowContext(ctx, `SELECT e.payload_policy,COALESCE(ws.custom_sensitive_fields,'[]'),SUM(CASE WHEN j.status IN ('pending','processing') THEN 1 ELSE 0 END) FROM events e JOIN delivery_jobs j ON j.event_id=e.id LEFT JOIN webhook_sources ws ON ws.id=e.source_id WHERE e.id=? GROUP BY e.id`, eventID).Scan(&policy, &sensitiveJSON, &remaining)
	var fields []string
	_ = json.Unmarshal([]byte(sensitiveJSON), &fields)
	return policy, fields, remaining == 0, err
}

func (s *Store) ReplaceEventPayload(ctx context.Context, eventID string, payload []byte) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE events SET payload_enc=? WHERE id=?`, payload, eventID)
	return err
}
func nullable(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
func (s *Store) RetryJob(ctx context.Context, id string) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE delivery_jobs SET status='pending',attempts=0,run_after=?,locked_at=NULL,last_error=NULL,updated_at=? WHERE id=?`, NowUnix(), NowUnix(), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) ListEvents(ctx context.Context, limit int) ([]Event, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,source_id,schedule_id,trigger_type,method,COALESCE(content_type,''),payload_policy,matched_rules,created_at FROM events ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var v Event
		var src, sch sql.NullString
		if err = rows.Scan(&v.ID, &src, &sch, &v.TriggerType, &v.Method, &v.ContentType, &v.PayloadPolicy, &v.MatchedRules, &v.CreatedAt); err != nil {
			return nil, err
		}
		if src.Valid {
			v.SourceID = &src.String
		}
		if sch.Valid {
			v.ScheduleID = &sch.String
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) ListDeliveries(ctx context.Context, limit int) ([]Delivery, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT j.id,j.event_id,j.status,j.attempts,j.run_after,j.last_error,c.name,t.name,j.created_at,j.updated_at FROM delivery_jobs j JOIN target_bindings b ON b.id=j.binding_id JOIN channels c ON c.id=b.channel_id JOIN message_templates t ON t.id=b.template_id ORDER BY j.created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Delivery
	for rows.Next() {
		var v Delivery
		if err = rows.Scan(&v.ID, &v.EventID, &v.Status, &v.Attempts, &v.RunAfter, &v.LastError, &v.ChannelName, &v.TemplateName, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) Dashboard(ctx context.Context) (map[string]any, error) {
	out := map[string]any{}
	for key, q := range map[string]string{"channels": `SELECT COUNT(*) FROM channels`, "sources": `SELECT COUNT(*) FROM webhook_sources`, "schedules": `SELECT COUNT(*) FROM schedules WHERE enabled=1`, "pending": `SELECT COUNT(*) FROM delivery_jobs WHERE status='pending'`, "failed": `SELECT COUNT(*) FROM delivery_jobs WHERE status='dead'`, "events_today": `SELECT COUNT(*) FROM events WHERE created_at>=strftime('%s','now','start of day')`} {
		var n int
		if err := s.DB.QueryRowContext(ctx, q).Scan(&n); err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		out[key] = n
	}
	return out, nil
}

func (s *Store) Cleanup(ctx context.Context, payloadDays, metadataDays int) error {
	_, _ = s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at<?`, NowUnix())
	_, err := s.DB.ExecContext(ctx, `UPDATE events SET payload_enc=NULL WHERE created_at<?`, time.Now().Add(-time.Duration(payloadDays)*24*time.Hour).Unix())
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `DELETE FROM events WHERE created_at<?`, time.Now().Add(-time.Duration(metadataDays)*24*time.Hour).Unix())
	return err
}
