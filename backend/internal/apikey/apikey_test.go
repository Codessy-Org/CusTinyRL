package apikey

import (
	"errors"
	"testing"
)

func TestValidateScopes(t *testing.T) {
	got, err := ValidateScopes([]string{"Create", " read ", "create"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "create" || got[1] != "read" {
		t.Fatalf("scopes = %v", got)
	}

	admin, err := ValidateScopes([]string{"admin"})
	if err != nil {
		t.Fatal(err)
	}
	if len(admin) != 3 {
		t.Fatalf("admin expands to %v", admin)
	}

	if _, err := ValidateScopes([]string{"nope"}); !errors.Is(err, ErrNoScope) {
		t.Fatalf("expected ErrNoScope, got %v", err)
	}
	if _, err := ValidateScopes(nil); err == nil {
		t.Fatal("empty scopes should fail")
	}
}

func TestLooksLikeKey(t *testing.T) {
	if !LooksLikeKey("ck_abc") {
		t.Fatal("ck_ prefix should match")
	}
	if LooksLikeKey("eyJhbGciOi") {
		t.Fatal("session tokens must not match")
	}
}

func TestValidateName(t *testing.T) {
	if _, err := validateName("ci pipeline 2"); err != nil {
		t.Fatalf("valid name rejected: %v", err)
	}
	if _, err := validateName(""); err == nil {
		t.Fatal("empty name should fail")
	}
	if _, err := validateName("bad\nname"); err == nil {
		t.Fatal("newline should fail")
	}
}
