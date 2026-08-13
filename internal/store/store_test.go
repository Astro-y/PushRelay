package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestIdempotencyWithinWindow(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	ctx := context.Background()
	source, err := s.SaveSource(ctx, Source{Name: "test", TokenHash: "hash", TokenPrefix: "token", MatchMode: "all_match", PayloadPolicy: "none", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	first, dup, err := s.AcceptEvent(ctx, AcceptEventInput{SourceID: source.ID, TriggerType: "webhook", IdempotencyKey: "same", Method: "POST", PayloadPolicy: "none"})
	if err != nil || dup {
		t.Fatal(first, dup, err)
	}
	second, dup, err := s.AcceptEvent(ctx, AcceptEventInput{SourceID: source.ID, TriggerType: "webhook", IdempotencyKey: "same", Method: "POST", PayloadPolicy: "none"})
	if err != nil || !dup || first != second {
		t.Fatal(first, second, dup, err)
	}
}

func TestRuntimeSettingsPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	defaults, err := s.RuntimeSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if defaults.DefaultTimezone != "Asia/Shanghai" || defaults.PayloadRetentionDays != 7 || defaults.MetadataRetentionDays != 30 || defaults.AllowPrivateWebhookTargets {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}
	want := RuntimeSettings{
		DefaultTimezone:            "Europe/Berlin",
		PayloadRetentionDays:       3,
		MetadataRetentionDays:      14,
		AllowPrivateWebhookTargets: true,
		PocketIDEnabled:            true,
		PocketIDIssuerURL:          "https://id.example.com",
		PocketIDClientID:           "client-id",
		PocketIDClientSecretEnc:    "encrypted-secret",
		PocketIDClientSecretSet:    true,
		PocketIDAllowedIdentity:    "admin@example.com",
	}
	if err = s.SaveRuntimeSettings(ctx, want); err != nil {
		t.Fatal(err)
	}
	if err = s.DB.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	got, err := s.RuntimeSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestAccountSecurityPersistence(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "security.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	ctx := context.Background()
	adminID, err := s.CreateAdmin(ctx, "admin", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetTOTPPending(ctx, adminID, []byte("encrypted")); err != nil {
		t.Fatal(err)
	}
	if err = s.EnableTOTP(ctx, adminID, []string{"code-1", "code-2"}); err != nil {
		t.Fatal(err)
	}
	admin, err := s.AdminSecurityByID(ctx, adminID)
	if err != nil || !admin.TOTPEnabled || string(admin.TOTPSecretEnc) != "encrypted" {
		t.Fatalf("unexpected admin security state: %+v, %v", admin, err)
	}
	used, err := s.ConsumeRecoveryCode(ctx, adminID, "code-1")
	if err != nil || !used {
		t.Fatal("expected recovery code to be consumed", err)
	}
	used, err = s.ConsumeRecoveryCode(ctx, adminID, "code-1")
	if err != nil || used {
		t.Fatal("recovery code was reusable", err)
	}
	if err = s.BindPocketIDSubject(ctx, adminID, "subject-1"); err != nil {
		t.Fatal(err)
	}
	admin, err = s.AdminSecurityByID(ctx, adminID)
	if err != nil || admin.PocketIDSubject != "subject-1" {
		t.Fatalf("unexpected Pocket ID binding: %+v, %v", admin, err)
	}
}
