package url

import (
	"strings"
	"testing"
)

func TestValidateCustomSlug(t *testing.T) {
	valid := map[string]string{
		"gauravresumev1": "gauravresumev1",
		"my-link_2":      "my-link_2",
		"a":              "a",
		"  spaced  ":     "spaced", // trims before validating
	}
	for in, want := range valid {
		got, err := ValidateCustomSlug(in)
		if err != nil {
			t.Fatalf("ValidateCustomSlug(%q) unexpected error: %v", in, err)
		}
		if got != want {
			t.Fatalf("ValidateCustomSlug(%q) = %q, want %q", in, got, want)
		}
	}

	invalid := []string{
		"",
		"UPPERCASE",
		"has space",
		"-leading",
		"_leading",
		"emoji😀",
		"dot.com",
		"slash/name",
		strings.Repeat("a", 65),
		"api",
		"API", // reserved check is case-insensitive
		"dashboard",
		"login",
		"health",
		"favicon.ico",
	}
	for _, in := range invalid {
		if _, err := ValidateCustomSlug(in); err == nil {
			t.Fatalf("ValidateCustomSlug(%q) expected error, got none", in)
		}
	}
}

func TestValidateRedirectSlug(t *testing.T) {
	ok := []string{"abc123", "a-b_c", "gA9xQ2Lm", strings.Repeat("z", 64)}
	for _, s := range ok {
		if _, valid := ValidateRedirectSlug(s); !valid {
			t.Fatalf("ValidateRedirectSlug(%q) should be valid", s)
		}
	}
	bad := []string{"", "has space", "emoji😀", strings.Repeat("z", 65), "a/b"}
	for _, s := range bad {
		if _, valid := ValidateRedirectSlug(s); valid {
			t.Fatalf("ValidateRedirectSlug(%q) should be invalid", s)
		}
	}
}

func TestValidatePrefix(t *testing.T) {
	if _, err := ValidatePrefix("resume"); err != nil {
		t.Fatalf("valid prefix rejected: %v", err)
	}
	for _, bad := range []string{"", "UPPER", "a b", strings.Repeat("x", 33)} {
		if _, err := ValidatePrefix(bad); err == nil {
			t.Fatalf("ValidatePrefix(%q) expected error", bad)
		}
	}
}

func TestGenerateSlug(t *testing.T) {
	s, err := GenerateSlug(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 10 {
		t.Fatalf("length = %d, want 10 (%q)", len(s), s)
	}
	for _, c := range s {
		if !strings.ContainsRune(base62Alphabet, c) {
			t.Fatalf("character %q not in base62 alphabet", c)
		}
	}
	// Uniqueness smoke: 10k 10-char base62 slugs should never collide.
	seen := make(map[string]bool, 10000)
	for i := 0; i < 10000; i++ {
		s, err := GenerateSlug(10)
		if err != nil {
			t.Fatal(err)
		}
		if seen[s] {
			t.Fatalf("duplicate slug generated: %q", s)
		}
		seen[s] = true
	}
	if _, err := GenerateSlug(0); err == nil {
		t.Fatal("GenerateSlug(0) should error")
	}
	if _, err := GenerateSlug(65); err == nil {
		t.Fatal("GenerateSlug(65) should error")
	}
}

func TestGenerateWithPrefix(t *testing.T) {
	s, err := GenerateWithPrefix("resume", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(s, "resume-") {
		t.Fatalf("missing prefix in %q", s)
	}
	if len(s) != len("resume-")+10 {
		t.Fatalf("unexpected length %d for %q", len(s), s)
	}
}

func TestIsReserved(t *testing.T) {
	for _, s := range []string{"api", "API", "dashboard", "_next"} {
		if !IsReserved(s) {
			t.Fatalf("%q should be reserved", s)
		}
	}
	if IsReserved("gauravresumev1") {
		t.Fatal("random slug should not be reserved")
	}
}

func BenchmarkGenerateSlug10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := GenerateSlug(10); err != nil {
			b.Fatal(err)
		}
	}
}
