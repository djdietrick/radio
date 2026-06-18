package auth

import (
	"testing"
	"time"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "s3cret" {
		t.Fatal("password must be hashed, not stored plaintext")
	}
	if !CheckPassword(hash, "s3cret") {
		t.Fatal("correct password should verify")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("wrong password should not verify")
	}
}

func TestIssueAndVerify(t *testing.T) {
	a := NewAuthenticator([]byte("test-secret"), time.Hour)
	tok, err := a.Issue("user-1", true)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := a.Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" || !claims.IsAdmin {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	issuer := NewAuthenticator([]byte("secret-a"), time.Hour)
	verifier := NewAuthenticator([]byte("secret-b"), time.Hour)
	tok, _ := issuer.Issue("user-1", false)
	if _, err := verifier.Verify(tok); err == nil {
		t.Fatal("token signed with a different secret must be rejected")
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	a := NewAuthenticator([]byte("secret"), -time.Minute) // already expired
	tok, _ := a.Issue("user-1", false)
	if _, err := a.Verify(tok); err == nil {
		t.Fatal("expired token must be rejected")
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	a := NewAuthenticator([]byte("secret"), time.Hour)
	if _, err := a.Verify("not.a.jwt"); err == nil {
		t.Fatal("garbage token must be rejected")
	}
}
