package store

const schema = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;

CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY);
INSERT OR IGNORE INTO schema_migrations(version) VALUES (1);

CREATE TABLE IF NOT EXISTS admins (
  id TEXT PRIMARY KEY, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL,
  created_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
  token_hash TEXT PRIMARY KEY, admin_id TEXT NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
  csrf_token TEXT NOT NULL, expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS channels (
  id TEXT PRIMARY KEY, name TEXT NOT NULL, type TEXT NOT NULL, config_enc BLOB NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS message_templates (
  id TEXT PRIMARY KEY, name TEXT NOT NULL, channel_type TEXT NOT NULL, body_json TEXT NOT NULL,
  created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS target_groups (
  id TEXT PRIMARY KEY, name TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS target_bindings (
  id TEXT PRIMARY KEY, group_id TEXT NOT NULL REFERENCES target_groups(id) ON DELETE CASCADE,
  channel_id TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
  template_id TEXT NOT NULL REFERENCES message_templates(id) ON DELETE CASCADE,
  enabled INTEGER NOT NULL DEFAULT 1,
  UNIQUE(group_id, channel_id, template_id)
);

CREATE TABLE IF NOT EXISTS webhook_sources (
  id TEXT PRIMARY KEY, name TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, token_prefix TEXT NOT NULL,
  hmac_secret_enc BLOB, allowed_cidrs TEXT NOT NULL DEFAULT '[]', match_mode TEXT NOT NULL DEFAULT 'all_match',
  custom_sensitive_fields TEXT NOT NULL DEFAULT '[]',
  payload_policy TEXT NOT NULL DEFAULT 'redacted', enabled INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS rules (
  id TEXT PRIMARY KEY, source_id TEXT NOT NULL REFERENCES webhook_sources(id) ON DELETE CASCADE,
  name TEXT NOT NULL, priority INTEGER NOT NULL DEFAULT 100, condition_json TEXT NOT NULL,
  target_group_id TEXT NOT NULL REFERENCES target_groups(id) ON DELETE CASCADE,
  enabled INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_rules_source ON rules(source_id, priority);

CREATE TABLE IF NOT EXISTS schedules (
  id TEXT PRIMARY KEY, name TEXT NOT NULL, recurrence_json TEXT NOT NULL, timezone TEXT NOT NULL,
  payload_enc BLOB NOT NULL, target_group_id TEXT NOT NULL REFERENCES target_groups(id) ON DELETE CASCADE,
  enabled INTEGER NOT NULL DEFAULT 1, next_run_at INTEGER, last_run_at INTEGER,
  created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_schedules_due ON schedules(enabled, next_run_at);

CREATE TABLE IF NOT EXISTS events (
  id TEXT PRIMARY KEY, source_id TEXT REFERENCES webhook_sources(id) ON DELETE SET NULL,
  schedule_id TEXT REFERENCES schedules(id) ON DELETE SET NULL, trigger_type TEXT NOT NULL,
  idempotency_key TEXT, method TEXT NOT NULL, content_type TEXT, payload_enc BLOB,
  payload_policy TEXT NOT NULL, matched_rules INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_idempotency ON events(source_id, idempotency_key, created_at) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_events_created ON events(created_at DESC);

CREATE TABLE IF NOT EXISTS delivery_jobs (
  id TEXT PRIMARY KEY, event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  binding_id TEXT NOT NULL REFERENCES target_bindings(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'pending', attempts INTEGER NOT NULL DEFAULT 0,
  run_after INTEGER NOT NULL, locked_at INTEGER, last_error TEXT,
  created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL,
  UNIQUE(event_id, binding_id)
);
CREATE INDEX IF NOT EXISTS idx_jobs_ready ON delivery_jobs(status, run_after);
CREATE TABLE IF NOT EXISTS delivery_attempts (
  id TEXT PRIMARY KEY, job_id TEXT NOT NULL REFERENCES delivery_jobs(id) ON DELETE CASCADE,
  attempt INTEGER NOT NULL, status TEXT NOT NULL, http_status INTEGER, duration_ms INTEGER NOT NULL,
  error TEXT, response_excerpt TEXT, created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_attempts_job ON delivery_attempts(job_id, created_at DESC);

CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at INTEGER NOT NULL
);
`

const migrationV2 = `
ALTER TABLE admins ADD COLUMN totp_secret_enc BLOB;
ALTER TABLE admins ADD COLUMN totp_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE admins ADD COLUMN pocketid_subject TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_admins_pocketid_subject ON admins(pocketid_subject) WHERE pocketid_subject IS NOT NULL;

CREATE TABLE IF NOT EXISTS admin_recovery_codes (
  code_hash TEXT PRIMARY KEY,
  admin_id TEXT NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS oauth_states (
  state_hash TEXT PRIMARY KEY,
  nonce TEXT NOT NULL,
  pkce_verifier TEXT NOT NULL,
  expires_at INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_oauth_states_expires ON oauth_states(expires_at);

INSERT OR REPLACE INTO schema_migrations(version) VALUES (2);
`
