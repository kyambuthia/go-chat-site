package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	if hash == password {
		t.Fatal("password should not be equal to hash")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	if !CheckPassword(password, hash) {
		t.Fatal("password should match hash")
	}

	if CheckPassword("wrongpassword", hash) {
		t.Fatal("wrong password should not match hash")
	}
}
