package store

import (
	"context"
	"strconv"
)

const (
	settingDefaultTimezone           = "default_timezone"
	settingPayloadRetentionDays      = "payload_retention_days"
	settingMetadataRetentionDays     = "metadata_retention_days"
	settingAllowPrivateWebhookTarget = "allow_private_webhook_targets"
	settingPocketIDEnabled           = "pocketid_enabled"
	settingPocketIDIssuerURL         = "pocketid_issuer_url"
	settingPocketIDClientID          = "pocketid_client_id"
	settingPocketIDClientSecret      = "pocketid_client_secret_enc"
	settingPocketIDAllowedIdentity   = "pocketid_allowed_identity"
)

type RuntimeSettings struct {
	DefaultTimezone            string `json:"default_timezone"`
	PayloadRetentionDays       int    `json:"payload_retention_days"`
	MetadataRetentionDays      int    `json:"metadata_retention_days"`
	AllowPrivateWebhookTargets bool   `json:"allow_private_webhook_targets"`
	PocketIDEnabled            bool   `json:"pocketid_enabled"`
	PocketIDIssuerURL          string `json:"pocketid_issuer_url"`
	PocketIDClientID           string `json:"pocketid_client_id"`
	PocketIDClientSecretEnc    string `json:"-"`
	PocketIDClientSecretSet    bool   `json:"pocketid_client_secret_configured"`
	PocketIDAllowedIdentity    string `json:"pocketid_allowed_identity"`
}

func DefaultRuntimeSettings() RuntimeSettings {
	return RuntimeSettings{
		DefaultTimezone:            "Asia/Shanghai",
		PayloadRetentionDays:       7,
		MetadataRetentionDays:      30,
		AllowPrivateWebhookTargets: false,
	}
}

func (s *Store) RuntimeSettings(ctx context.Context) (RuntimeSettings, error) {
	settings := DefaultRuntimeSettings()
	rows, err := s.DB.QueryContext(ctx, `SELECT key,value FROM settings WHERE key IN (?,?,?,?,?,?,?,?,?)`,
		settingDefaultTimezone,
		settingPayloadRetentionDays,
		settingMetadataRetentionDays,
		settingAllowPrivateWebhookTarget,
		settingPocketIDEnabled,
		settingPocketIDIssuerURL,
		settingPocketIDClientID,
		settingPocketIDClientSecret,
		settingPocketIDAllowedIdentity,
	)
	if err != nil {
		return RuntimeSettings{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err = rows.Scan(&key, &value); err != nil {
			return RuntimeSettings{}, err
		}
		switch key {
		case settingDefaultTimezone:
			settings.DefaultTimezone = value
		case settingPayloadRetentionDays:
			if parsed, parseErr := strconv.Atoi(value); parseErr == nil {
				settings.PayloadRetentionDays = parsed
			}
		case settingMetadataRetentionDays:
			if parsed, parseErr := strconv.Atoi(value); parseErr == nil {
				settings.MetadataRetentionDays = parsed
			}
		case settingAllowPrivateWebhookTarget:
			if parsed, parseErr := strconv.ParseBool(value); parseErr == nil {
				settings.AllowPrivateWebhookTargets = parsed
			}
		case settingPocketIDEnabled:
			if parsed, parseErr := strconv.ParseBool(value); parseErr == nil {
				settings.PocketIDEnabled = parsed
			}
		case settingPocketIDIssuerURL:
			settings.PocketIDIssuerURL = value
		case settingPocketIDClientID:
			settings.PocketIDClientID = value
		case settingPocketIDClientSecret:
			settings.PocketIDClientSecretEnc = value
			settings.PocketIDClientSecretSet = value != ""
		case settingPocketIDAllowedIdentity:
			settings.PocketIDAllowedIdentity = value
		}
	}
	return settings, rows.Err()
}

func (s *Store) SaveRuntimeSettings(ctx context.Context, settings RuntimeSettings) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	values := map[string]string{
		settingDefaultTimezone:           settings.DefaultTimezone,
		settingPayloadRetentionDays:      strconv.Itoa(settings.PayloadRetentionDays),
		settingMetadataRetentionDays:     strconv.Itoa(settings.MetadataRetentionDays),
		settingAllowPrivateWebhookTarget: strconv.FormatBool(settings.AllowPrivateWebhookTargets),
		settingPocketIDEnabled:           strconv.FormatBool(settings.PocketIDEnabled),
		settingPocketIDIssuerURL:         settings.PocketIDIssuerURL,
		settingPocketIDClientID:          settings.PocketIDClientID,
		settingPocketIDClientSecret:      settings.PocketIDClientSecretEnc,
		settingPocketIDAllowedIdentity:   settings.PocketIDAllowedIdentity,
	}
	for key, value := range values {
		if _, err = tx.ExecContext(ctx, `INSERT INTO settings(key,value,updated_at) VALUES(?,?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value,updated_at=excluded.updated_at`, key, value, NowUnix()); err != nil {
			return err
		}
	}
	return tx.Commit()
}
