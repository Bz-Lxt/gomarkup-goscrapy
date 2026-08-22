package auth

import (
	"testing"
	"time"
)

func TestAuthenticate(t *testing.T) {
	s := New("secret", time.Hour)
	_, _, err := s.Authenticate("admin", "wrong")
	if err != ErrBadCredential {
		t.Fatalf("want bad cred, got %v", err)
	}
	tok, exp, err := s.Authenticate("admin", "Admin@12345")
	if err != nil || tok == "" || exp != 3600 {
		t.Fatalf("login %v exp=%d", err, exp)
	}
	c, err := s.Parse(tok)
	if err != nil || c.Username != "admin" {
		t.Fatalf("parse %v %+v", err, c)
	}
	if _, err := s.Parse("nope"); err != ErrUnauthorized {
		t.Fatalf("bad token %v", err)
	}
}
