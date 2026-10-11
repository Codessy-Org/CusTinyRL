package preview

import (
	"context"
	"net"
	"testing"
)

func TestParseTitleAndIcon(t *testing.T) {
	html := `<html><head>
		<meta charset="utf-8">
		<title>  My   Cool
		Page  </title>
		<link rel="stylesheet" href="/style.css">
		<link rel="icon" type="image/png" href="/favicons/icon.png">
	</head><body></body></html>`
	got := parse(html, "https://example.com/jobs/123")
	if got.Title != "My Cool Page" {
		t.Fatalf("title = %q", got.Title)
	}
	if got.FaviconURL != "https://example.com/favicons/icon.png" {
		t.Fatalf("favicon = %q", got.FaviconURL)
	}
}

func TestParseFallbackFavicon(t *testing.T) {
	got := parse(`<title>X</title>`, "https://example.com/a/b")
	if got.FaviconURL != "https://example.com/favicon.ico" {
		t.Fatalf("favicon = %q", got.FaviconURL)
	}
}

func TestParseTitleTruncated(t *testing.T) {
	long := make([]byte, 300)
	for i := range long {
		long[i] = 'a'
	}
	got := parse("<title>"+string(long)+"</title>", "https://example.com")
	if len(got.Title) != 200 {
		t.Fatalf("title len = %d, want 200", len(got.Title))
	}
}

func TestResolveAbsoluteIconWins(t *testing.T) {
	got := parse(`<link rel="apple-touch-icon" href="https://cdn.example.com/touch.png">`, "https://example.com")
	if got.FaviconURL != "https://cdn.example.com/touch.png" {
		t.Fatalf("favicon = %q", got.FaviconURL)
	}
}

func TestFetchRejectsPrivateAddresses(t *testing.T) {
	ctx := context.Background()
	for _, u := range []string{
		"http://127.0.0.1/x",
		"http://localhost/x",
		"http://10.0.0.5/x",
		"http://192.168.1.1/x",
		"http://169.254.169.254/latest/meta-data", // cloud metadata
		"http://[::1]/x",
		"file:///etc/passwd",
	} {
		if _, err := Fetch(ctx, u); err == nil {
			t.Fatalf("expected rejection for %s", u)
		}
	}
}

func TestIsPublicIP(t *testing.T) {
	public := net.ParseIP("93.184.216.34")
	if !isPublicIP(public) {
		t.Fatal("example.com IP should be public")
	}
	for _, s := range []string{"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.0.1", "169.254.1.1", "100.64.0.1", "::1", "fe80::1"} {
		if isPublicIP(net.ParseIP(s)) {
			t.Fatalf("%s should not be public", s)
		}
	}
}
