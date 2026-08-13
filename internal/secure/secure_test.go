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

func TestRecoveryCode(t *testing.T) {
	code, err := RecoveryCode()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 19 || len(NormalizeRecoveryCode(code)) != 16 {
		t.Fatalf("unexpected recovery code format: %q", code)
	}
}
