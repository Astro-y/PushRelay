package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"time"

	"pushrelay/internal/rules"
	"pushrelay/internal/schedule"
	"pushrelay/internal/store"
	"pushrelay/internal/templatex"
)

func (s *Server) worker(ctx context.Context, index int) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for i := 0; i < 8; i++ {
				job, err := s.store.ClaimJob(ctx)
				if errors.Is(err, sql.ErrNoRows) {
					break
				}
				if err != nil {
					s.logger.Error("claim delivery job", "worker", index, "error", err)
					break
				}
				s.processJob(ctx, job)
			}
		}
	}
}
func (s *Server) processJob(ctx context.Context, job store.JobPayload) {
	started := time.Now()
	attempt := job.Attempts + 1
	httpStatus := 0
	excerpt := ""
	var sendErr error
	if !job.Channel.Enabled {
		sendErr = errors.New("channel is disabled")
	} else {
		payload, err := s.vault.Decrypt(job.PayloadEnc)
		if err != nil {
			sendErr = err
		} else {
			var envelope map[string]any
			if err = json.Unmarshal(payload, &envelope); err != nil {
				sendErr = err
			} else {
				rendered, err := templatex.Render(job.Template.Body, envelope)
				if err != nil {
					sendErr = err
				} else {
					cfgRaw, err := s.vault.Decrypt(job.Channel.ConfigEnc)
					if err != nil {
						sendErr = err
					} else {
						var cfg map[string]any
						if err = json.Unmarshal(cfgRaw, &cfg); err != nil {
							sendErr = err
						} else {
							sendCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
							httpStatus, excerpt, sendErr = s.channels.Send(sendCtx, job.Channel.Type, cfg, rendered)
							cancel()
						}
					}
				}
			}
		}
	}
	errText := ""
	if sendErr != nil {
		errText = sendErr.Error()
	}
	if len(excerpt) > 1024 {
		excerpt = excerpt[:1024]
	}
	if err := s.store.FinishAttempt(ctx, job.JobID, attempt, sendErr == nil, httpStatus, int(time.Since(started).Milliseconds()), errText, excerpt); err != nil {
		s.logger.Error("finish delivery attempt", "job_id", job.JobID, "error", err)
	} else {
		s.logger.Info("delivery completed", "job_id", job.JobID, "attempt", attempt, "success", sendErr == nil, "http_status", httpStatus)
		s.finalizeEventPayload(ctx, job.EventID, job.PayloadEnc)
	}
}

func (s *Server) finalizeEventPayload(ctx context.Context, eventID string, original []byte) {
	policy, sensitiveFields, complete, err := s.store.EventPayloadPolicyIfComplete(ctx, eventID)
	if err != nil || !complete || policy == "redacted" && len(original) == 0 {
		return
	}
	if policy == "none" || policy == "metadata" {
		_ = s.store.ReplaceEventPayload(ctx, eventID, nil)
		return
	}
	plain, err := s.vault.Decrypt(original)
	if err != nil {
		return
	}
	var value any
	if json.Unmarshal(plain, &value) != nil {
		return
	}
	redacted, _ := json.Marshal(rules.RedactWithFields(value, sensitiveFields))
	enc, err := s.vault.Encrypt(redacted)
	if err == nil {
		_ = s.store.ReplaceEventPayload(ctx, eventID, enc)
	}
}

func (s *Server) scheduler(ctx context.Context) {
	s.runDueSchedules(ctx)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runDueSchedules(ctx)
		}
	}
}
func (s *Server) runDueSchedules(ctx context.Context) {
	items, err := s.store.DueSchedules(ctx, time.Now().Unix())
	if err != nil {
		s.logger.Error("load due schedules", "error", err)
		return
	}
	for _, item := range items {
		payload, err := s.vault.Decrypt(item.PayloadEnc)
		if err != nil {
			s.logger.Error("decrypt schedule payload", "schedule_id", item.ID, "error", err)
			continue
		}
		var body any
		if err = json.Unmarshal(payload, &body); err != nil {
			s.logger.Error("parse schedule payload", "schedule_id", item.ID, "error", err)
			continue
		}
		eventID := store.NewID()
		envelope := map[string]any{"body": body, "raw_body": string(payload), "headers": map[string]any{}, "query": map[string]any{}, "form": map[string]any{}, "meta": map[string]any{"event_id": eventID, "received_at": time.Now().UTC().Format(time.RFC3339), "method": "SCHEDULE", "schedule_id": item.ID, "schedule_name": item.Name}}
		raw, _ := json.Marshal(envelope)
		enc, err := s.vault.Encrypt(raw)
		if err != nil {
			continue
		}
		_, _, err = s.store.AcceptEvent(ctx, store.AcceptEventInput{ID: eventID, ScheduleID: item.ID, TriggerType: "schedule", Method: "SCHEDULE", ContentType: "application/json", PayloadPolicy: "redacted", PayloadEnc: enc, MatchedRules: 1, GroupIDs: []string{item.TargetGroupID}})
		if err != nil {
			s.logger.Error("enqueue schedule", "schedule_id", item.ID, "error", err)
			continue
		}
		next, err := schedule.Next(item.Recurrence, item.Timezone, time.Now())
		if err != nil {
			s.logger.Error("advance schedule", "schedule_id", item.ID, "error", err)
			continue
		}
		var nextUnix *int64
		if next != nil {
			v := next.Unix()
			nextUnix = &v
		}
		if err = s.store.MarkScheduleRun(ctx, item.ID, time.Now().Unix(), nextUnix); err != nil {
			s.logger.Error("mark schedule", "schedule_id", item.ID, "error", err)
		}
	}
}

func (s *Server) maintenance(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			settings := s.currentSettings()
			if err := s.store.Cleanup(ctx, settings.PayloadRetentionDays, settings.MetadataRetentionDays); err != nil {
				s.logger.Error("cleanup retention", "error", err)
			}
		}
	}
}

var _ = rand.IntN
