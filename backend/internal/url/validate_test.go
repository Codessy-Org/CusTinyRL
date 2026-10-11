package url

import "testing"

func TestValidateDestinationURL(t *testing.T) {
	valid := []string{
		"https://docs.google.com/document/d/example-document-id",
		"http://example.com/path?q=1",
		"https://example.com",
		"https://sub.domain.example.org/a/b/c#frag", // fragment stripped
	}
	for _, in := range valid {
		got, err := ValidateDestinationURL(in)
		if err != nil {
			t.Fatalf("ValidateDestinationURL(%q) unexpected error: %v", in, err)
		}
		if got == "" {
			t.Fatalf("empty result for %q", in)
		}
	}

	invalid := []string{
		"",
		"javascript:alert(1)",
		"data:text/html,hello",
		"file:///etc/passwd",
		"ftp://example.com",
		"https://user:pass@example.com",
		"not a url",
		"https://",
	}
	for _, in := range invalid {
		if _, err := ValidateDestinationURL(in); err == nil {
			t.Fatalf("ValidateDestinationURL(%q) expected error", in)
		}
	}
}

func TestValidateDestinationURLStripsFragment(t *testing.T) {
	got, err := ValidateDestinationURL("https://example.com/page#section")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://example.com/page" {
		t.Fatalf("got %q", got)
	}
}
