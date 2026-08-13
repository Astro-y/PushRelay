package secure

import "testing"

func TestVaultRoundTrip(t *testing.T) {
	v, err := NewVault(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	enc, err := v.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := v.Decrypt(enc)
	if err != nil || string(plain) != "secret" {
		t.Fatal(string(plain), err)
	}
}
func TestPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(hash, "correct horse battery staple") || VerifyPassword(hash, "wrong password") {
		t.Fatal("password verification failed")
	}
}

func TestPasswordMinimumLength(t *testing.T) {
	if _, err := HashPassword("1234567"); err == nil {
		t.Fatal("HashPassword() accepted a password shorter than 8 characters")
	}
	if _, err := HashPassword("一二三四五六七"); err == nil {
		t.Fatal("HashPassword() counted UTF-8 bytes instead of characters")
	}
	if _, err := HashPassword("12345678"); err != nil {
		t.Fatalf("HashPassword() rejected an 8-character password: %v", err)
	}
}

func TestRecoveryCode(t *testing.T) {
	code, err := RecoveryCode()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 19 || len(NormalizeRecoveryCode(code)) != 16 {
		t.Fatalf("unexpected recovery code format: %q", code)
	}
}
