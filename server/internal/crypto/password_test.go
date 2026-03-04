package crypto

import "testing"

func TestHashPasswordAndCheckPasswordHash(t *testing.T) {
	hash, err := HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if !CheckPasswordHash("password123", hash) {
		t.Fatal("expected password to match hash")
	}
	if CheckPasswordHash("wrong-password", hash) {
		t.Fatal("expected wrong password to fail hash check")
	}
}
