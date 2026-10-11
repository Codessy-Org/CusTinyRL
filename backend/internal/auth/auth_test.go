package auth

import (
	"strings"
	"testing"
)

func TestHashTokenDeterministicAndUnique(t *testing.T) {
	a := hashToken("token-a")
	b := hashToken("token-a")
	c := hashToken("token-b")
	if string(a) != string(b) {
		t.Fatal("hash must be deterministic")
	}
	if string(a) == string(c) {
		t.Fatal("different tokens must hash differently")
	}
	if len(a) != 32 {
		t.Fatalf("sha256 digest length = %d", len(a))
	}
}

func TestNewTokenStringEntropy(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		s, err := newTokenString()
		if err != nil {
			t.Fatal(err)
		}
		if len(s) < 40 {
			t.Fatalf("token too short: %q", s)
		}
		if strings.ContainsAny(s, "+/=") {
			t.Fatalf("token not URL-safe: %q", s)
		}
		if seen[s] {
			t.Fatalf("duplicate token: %q", s)
		}
		seen[s] = true
	}
}

func TestValidateEmail(t *testing.T) {
	for _, ok := range []string{"a@b.co", "user.name+tag@example.com"} {
		if err := validateEmail(ok); err != nil {
			t.Fatalf("validateEmail(%q) = %v", ok, err)
		}
	}
	for _, bad := range []string{"", "no-at", "@nope", "a@", strings.Repeat("x", 255) + "@y.z"} {
		if err := validateEmail(bad); err == nil {
			t.Fatalf("validateEmail(%q) expected error", bad)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := normalizeEmail("  Foo@Bar.COM "); got != "foo@bar.com" {
		t.Fatalf("got %q", got)
	}
}
