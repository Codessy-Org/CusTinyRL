// Package httpx helpers: trusted-proxy aware client IP resolution.
package httpx

import (
	"net"
	"net/http"
	"strings"
)

// IPResolver extracts the real client IP, trusting X-Forwarded-For only when
// the direct peer is inside the configured trusted proxy CIDRs.
type IPResolver struct {
	trusted []*net.IPNet
}

func NewIPResolver(cidrs []string) (*IPResolver, error) {
	r := &IPResolver{}
	for _, c := range cidrs {
		if !strings.Contains(c, "/") {
			if ip := net.ParseIP(c); ip != nil {
				if ip.To4() != nil {
					c += "/32"
				} else {
					c += "/128"
				}
			}
		}
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			return nil, err
		}
		r.trusted = append(r.trusted, n)
	}
	return r, nil
}

func (r *IPResolver) isTrusted(ip net.IP) bool {
	for _, n := range r.trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIP returns the best-effort client address for rate limiting and logs.
func (r *IPResolver) ClientIP(req *http.Request) string {
	remote := req.RemoteAddr
	if host, _, err := net.SplitHostPort(remote); err == nil {
		remote = host
	}
	rip := net.ParseIP(remote)
	if rip == nil || !r.isTrusted(rip) {
		return remote
	}
	// Walk X-Forwarded-For right to left, skipping trusted proxies.
	hops := strings.Split(req.Header.Get("X-Forwarded-For"), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		h := strings.TrimSpace(hops[i])
		if h == "" {
			continue
		}
		ip := net.ParseIP(h)
		if ip == nil {
			break
		}
		if !r.isTrusted(ip) {
			return ip.String()
		}
	}
	if f := strings.TrimSpace(req.Header.Get("X-Real-IP")); f != "" {
		if ip := net.ParseIP(f); ip != nil && !r.isTrusted(ip) {
			return ip.String()
		}
	}
	return remote
}
