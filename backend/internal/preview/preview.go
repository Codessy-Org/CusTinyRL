// Package preview fetches page title and favicon for a destination URL so
// the dashboard can show meaningful link previews. Fetches are bounded
// (timeout, body size) and SSRF-guarded: private, loopback, and link-local
// addresses are refused.
package preview

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Result is the (best-effort) metadata discovered for a URL.
type Result struct {
	Title      string
	FaviconURL string
}

var (
	titleRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	// rel="icon"-ish link tags; captures href in either attribute order.
	iconRe = regexp.MustCompile(`(?is)<link\b[^>]*rel=["'][^"']*(?:icon|shortcut icon|apple-touch-icon)[^"']*["'][^>]*>`)
	hrefRe = regexp.MustCompile(`(?is)href=["']([^"']+)["']`)
)

const (
	maxBody = 512 << 10 // 512 KiB
	timeout = 3 * time.Second
)

// Fetch retrieves title and favicon for rawURL. Errors are non-fatal for
// callers: any failure returns a zero Result and an error.
func Fetch(ctx context.Context, rawURL string) (Result, error) {
	if err := checkPublicURL(rawURL); err != nil {
		return Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", "cut.sh-preview/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	client := &http.Client{
		// Redirects are followed but each hop is re-checked by CheckRedirect.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			return checkPublicURL(req.URL.String())
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(strings.ToLower(ct), "html") {
		return Result{}, errors.New("not an html document")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return Result{}, err
	}
	return parse(string(body), rawURL), nil
}

func parse(html, base string) Result {
	var out Result
	if m := titleRe.FindStringSubmatch(html); len(m) == 2 {
		t := strings.TrimSpace(m[1])
		t = strings.Join(strings.Fields(t), " ")
		if len(t) > 200 {
			t = t[:200]
		}
		out.Title = t
	}
	if tag := iconRe.FindString(html); tag != "" {
		if m := hrefRe.FindStringSubmatch(tag); len(m) == 2 {
			out.FaviconURL = resolveURL(base, strings.TrimSpace(m[1]))
		}
	}
	if out.FaviconURL == "" {
		if u, err := url.Parse(base); err == nil {
			out.FaviconURL = strings.TrimRight(u.Scheme+"://"+u.Host, "/") + "/favicon.ico"
		}
	}
	return out
}

func resolveURL(base, ref string) string {
	b, err := url.Parse(base)
	if err != nil {
		return ""
	}
	r, err := url.Parse(ref)
	if err != nil || r.IsAbs() {
		if err != nil {
			return ""
		}
		return r.String()
	}
	return b.ResolveReference(r).String()
}

// checkPublicURL rejects non-http(s) schemes and URLs whose host resolves to
// a private, loopback, link-local, or otherwise non-routable address.
func checkPublicURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("invalid url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("only http(s) urls are previewed")
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("missing host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return errors.New("private address")
		}
		return nil
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return fmt.Errorf("resolve host: %w", err)
	}
	if len(addrs) == 0 {
		return errors.New("host does not resolve")
	}
	for _, a := range addrs {
		if !isPublicIP(a.IP) {
			return errors.New("private address")
		}
	}
	return nil
}

func isPublicIP(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	// Cloud metadata and CGNAT ranges.
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 { // 169.254.0.0/16
			return false
		}
		if ip4[0] == 100 && ip4[1]&0xc0 == 64 { // 100.64.0.0/10
			return false
		}
	}
	return true
}
