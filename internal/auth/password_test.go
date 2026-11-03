package auth

import "testing"

func TestArgon2idPasswordWithPepper(t *testing.T) {
	const pepper = "development-test-pepper-with-32-characters"
	hash, err := HashPassword("RahasiaAman123", pepper)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword(hash, "RahasiaAman123", pepper)
	if err != nil || !ok {
		t.Fatalf("password benar ditolak: ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword(hash, "PasswordSalah123", pepper)
	if err != nil || ok {
		t.Fatalf("password salah diterima: ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword(hash, "RahasiaAman123", pepper+"-beda")
	if err != nil || ok {
		t.Fatalf("pepper berbeda diterima: ok=%v err=%v", ok, err)
	}
}
