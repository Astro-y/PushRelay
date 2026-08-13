package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct{ DB *sql.DB }

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if err = migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{DB: db}, nil
}

func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		return err
	}
	if version >= 2 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(migrationV2); err != nil {
		return err
	}
	return tx.Commit()
}

func NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func NowUnix() int64 { return time.Now().UTC().Unix() }

func (s *Store) HasAdmin(ctx context.Context) (bool, error) {
	var count int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins`).Scan(&count)
	return count > 0, err
}

func (s *Store) CreateAdmin(ctx context.Context, username, passwordHash string) (string, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		return "", err
	}
	if count > 0 {
		return "", errors.New("administrator already exists")
	}
	id := NewID()
	if _, err = tx.ExecContext(ctx, `INSERT INTO admins(id,username,password_hash,created_at) VALUES(?,?,?,?)`, id, username, passwordHash, NowUnix()); err != nil {
		return "", err
	}
	return id, tx.Commit()
}

func (s *Store) AdminByUsername(ctx context.Context, username string) (id, hash string, err error) {
	err = s.DB.QueryRowContext(ctx, `SELECT id,password_hash FROM admins WHERE username=?`, username).Scan(&id, &hash)
	return
}

type AdminSecurity struct {
	ID              string
	Username        string
	PasswordHash    string
	TOTPSecretEnc   []byte
	TOTPEnabled     bool
	PocketIDSubject string
}

func (s *Store) AdminSecurityByUsername(ctx context.Context, username string) (admin AdminSecurity, err error) {
	var enabled int
	var subject sql.NullString
	err = s.DB.QueryRowContext(ctx, `SELECT id,username,password_hash,totp_secret_enc,totp_enabled,pocketid_subject FROM admins WHERE username=?`, username).
		Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &admin.TOTPSecretEnc, &enabled, &subject)
	admin.TOTPEnabled = enabled != 0
	admin.PocketIDSubject = subject.String
	return
}

func (s *Store) AdminSecurityByID(ctx context.Context, id string) (admin AdminSecurity, err error) {
	var enabled int
	var subject sql.NullString
	err = s.DB.QueryRowContext(ctx, `SELECT id,username,password_hash,totp_secret_enc,totp_enabled,pocketid_subject FROM admins WHERE id=?`, id).
		Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &admin.TOTPSecretEnc, &enabled, &subject)
	admin.TOTPEnabled = enabled != 0
	admin.PocketIDSubject = subject.String
	return
}

func (s *Store) SoleAdminSecurity(ctx context.Context) (admin AdminSecurity, err error) {
	var enabled int
	var subject sql.NullString
	err = s.DB.QueryRowContext(ctx, `SELECT id,username,password_hash,totp_secret_enc,totp_enabled,pocketid_subject FROM admins ORDER BY created_at LIMIT 1`).
		Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &admin.TOTPSecretEnc, &enabled, &subject)
	admin.TOTPEnabled = enabled != 0
	admin.PocketIDSubject = subject.String
	return
}

func (s *Store) UpdateUsername(ctx context.Context, adminID, username string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE admins SET username=? WHERE id=?`, username, adminID)
	return err
}

func (s *Store) UpdatePassword(ctx context.Context, adminID, passwordHash, currentSessionHash string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE admins SET password_hash=? WHERE id=?`, passwordHash, adminID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE admin_id=? AND token_hash<>?`, adminID, currentSessionHash); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SetTOTPPending(ctx context.Context, adminID string, secretEnc []byte) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE admins SET totp_secret_enc=?,totp_enabled=0 WHERE id=?`, secretEnc, adminID)
	return err
}

func (s *Store) EnableTOTP(ctx context.Context, adminID string, recoveryCodeHashes []string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE admins SET totp_enabled=1 WHERE id=? AND totp_secret_enc IS NOT NULL`, adminID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM admin_recovery_codes WHERE admin_id=?`, adminID); err != nil {
		return err
	}
	for _, codeHash := range recoveryCodeHashes {
		if _, err = tx.ExecContext(ctx, `INSERT INTO admin_recovery_codes(code_hash,admin_id,created_at) VALUES(?,?,?)`, codeHash, adminID, NowUnix()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) DisableTOTP(ctx context.Context, adminID string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE admins SET totp_secret_enc=NULL,totp_enabled=0 WHERE id=?`, adminID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM admin_recovery_codes WHERE admin_id=?`, adminID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ConsumeRecoveryCode(ctx context.Context, adminID, codeHash string) (bool, error) {
	result, err := s.DB.ExecContext(ctx, `DELETE FROM admin_recovery_codes WHERE admin_id=? AND code_hash=?`, adminID, codeHash)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func (s *Store) BindPocketIDSubject(ctx context.Context, adminID, subject string) error {
	result, err := s.DB.ExecContext(ctx, `UPDATE admins SET pocketid_subject=? WHERE id=? AND (pocketid_subject IS NULL OR pocketid_subject='' OR pocketid_subject=?)`, subject, adminID, subject)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return errors.New("Pocket ID identity is already bound to another subject")
	}
	return nil
}

type OAuthState struct {
	Nonce        string
	PKCEVerifier string
}

func (s *Store) CreateOAuthState(ctx context.Context, stateHash, nonce, verifier string, expires time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM oauth_states WHERE expires_at<=?`, NowUnix()); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO oauth_states(state_hash,nonce,pkce_verifier,expires_at,created_at) VALUES(?,?,?,?,?)`, stateHash, nonce, verifier, expires.Unix(), NowUnix()); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ConsumeOAuthState(ctx context.Context, stateHash string) (OAuthState, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return OAuthState{}, err
	}
	defer tx.Rollback()
	var state OAuthState
	var expires int64
	if err = tx.QueryRowContext(ctx, `SELECT nonce,pkce_verifier,expires_at FROM oauth_states WHERE state_hash=?`, stateHash).Scan(&state.Nonce, &state.PKCEVerifier, &expires); err != nil {
		return OAuthState{}, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM oauth_states WHERE state_hash=?`, stateHash); err != nil {
		return OAuthState{}, err
	}
	if expires <= NowUnix() {
		return OAuthState{}, errors.New("OAuth state expired")
	}
	return state, tx.Commit()
}

func (s *Store) CreateSession(ctx context.Context, tokenHash, adminID, csrf string, expires time.Time) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO sessions(token_hash,admin_id,csrf_token,expires_at,created_at) VALUES(?,?,?,?,?)`, tokenHash, adminID, csrf, expires.Unix(), NowUnix())
	return err
}

func (s *Store) Session(ctx context.Context, tokenHash string) (adminID, username, csrf string, expires int64, err error) {
	err = s.DB.QueryRowContext(ctx, `SELECT a.id,a.username,s.csrf_token,s.expires_at FROM sessions s JOIN admins a ON a.id=s.admin_id WHERE s.token_hash=? AND s.expires_at>?`, tokenHash, NowUnix()).Scan(&adminID, &username, &csrf, &expires)
	return
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=?`, tokenHash)
	return err
}

func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }
