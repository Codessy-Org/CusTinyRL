package url

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

const maxDestinationURLLen = 2048

// ValidateDestinationURL accepts only absolute http(s) URLs without embedded
// credentials. Loopback, private, and link-local hosts are allowed as
// destinations (linking to an internal dashboard is legitimate) — SSRF
// protections apply only to server-side fetchers, which this service does not
// implement.
func ValidateDestinationURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("url is required")
	}
	if len(raw) > maxDestinationURLLen {
		return "", fmt.Errorf("url must be at most %d characters", maxDestinationURLLen)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("url is malformed")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("only http and https urls are allowed")
	}
	if u.User != nil {
		return "", errors.New("urls with embedded credentials are not allowed")
	}
	if u.Host == "" {
		return "", errors.New("url must include a host")
	}
	if _, _, err := net.SplitHostPort(u.Host); err != nil && strings.Contains(u.Host, "@") {
		return "", errors.New("url host is malformed")
	}
	u.Fragment = ""
	return u.String(), nil
}
