package url

import (
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	base62Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	maxSlugLen     = 64
	maxPrefixLen   = 32
)

var (
	customSlugRe   = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	prefixRe       = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
	redirectSlugRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)
)

// ErrSlugTaken reports that a requested custom slug already exists.
var ErrSlugTaken = errors.New("slug already exists")

// ErrAuthRequired reports that the operation (custom slug / prefix) needs a
// signed-in user.
var ErrAuthRequired = errors.New("authentication required")

// reservedSlugs collide with application routes or common well-known paths.
var reservedSlugs = map[string]bool{
	"api": true, "app": true, "admin": true, "auth": true,
	"callback": true, "dashboard": true, "docs": true, "favicon.ico": true,
	"health": true, "healthz": true, "help": true, "live": true,
	"login": true, "logout": true, "me": true, "metrics": true,
	"oauth": true, "privacy": true, "profile": true, "public": true,
	"ready": true, "redirect": true, "register": true, "robots.txt": true,
	"settings": true, "signup": true, "sitemap.xml": true, "static": true,
	"status": true, "terms": true, "urls": true, "v1": true, "www": true,
	"_next": true,
}

// IsReserved reports whether the canonical slug collides with a protected path.
func IsReserved(slug string) bool { return reservedSlugs[strings.ToLower(slug)] }

// ValidateCustomSlug validates and canonicalizes a user-chosen slug.
// Custom slugs are restricted to lowercase so canonical IDs never
// duplicate via case (redirect lookup remains case-sensitive because
// generated Base62 slugs are mixed-case).
func ValidateCustomSlug(slug string) (string, error) {
	s := strings.TrimSpace(slug)
	if s == "" {
		return "", errors.New("slug is required")
	}
	if len(s) > maxSlugLen {
		return "", fmt.Errorf("slug must be at most %d characters", maxSlugLen)
	}
	if s != strings.ToLower(s) {
		return "", errors.New("custom slugs must be lowercase")
	}
	if !customSlugRe.MatchString(s) {
		return "", errors.New("slug may only contain lowercase letters, numbers, hyphens, and underscores")
	}
	if IsReserved(s) {
		return "", fmt.Errorf("slug %q is reserved", s)
	}
	return s, nil
}

// ValidateRedirectSlug validates a slug coming from the public redirect path.
func ValidateRedirectSlug(slug string) (string, bool) {
	if slug == "" || len(slug) > maxSlugLen {
		return "", false
	}
	if !redirectSlugRe.MatchString(slug) {
		return "", false
	}
	return slug, true
}

// ValidatePrefix validates an optional prefix for generated slugs.
func ValidatePrefix(prefix string) (string, error) {
	p := strings.TrimSpace(prefix)
	if p == "" {
		return "", errors.New("prefix is required")
	}
	if len(p) > maxPrefixLen {
		return "", fmt.Errorf("prefix must be at most %d characters", maxPrefixLen)
	}
	if p != strings.ToLower(p) {
		return "", errors.New("prefix must be lowercase")
	}
	if !prefixRe.MatchString(p) {
		return "", errors.New("prefix may only contain lowercase letters, numbers, hyphens, and underscores")
	}
	return p, nil
}

// GenerateSlug returns a cryptographically secure random Base62 slug of
// exactly n characters. Characters are drawn with rejection sampling so the
// distribution is uniform; uniqueness is enforced by the database unique
// constraint (with bounded retries in the service layer), never by the
// generator alone.
func GenerateSlug(n int) (string, error) {
	if n < 1 || n > maxSlugLen {
		return "", fmt.Errorf("slug length must be between 1 and %d", maxSlugLen)
	}
	// Largest multiple of 62 below 256 to keep sampling unbiased.
	const unbiasedMax = 248 // 62 * 4
	out := make([]byte, n)
	for i := 0; i < n; {
		buf := make([]byte, (n-i)*2)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("read random: %w", err)
		}
		for _, b := range buf {
			if b >= unbiasedMax {
				continue
			}
			out[i] = base62Alphabet[int(b)%62]
			i++
			if i == n {
				break
			}
		}
	}
	return string(out), nil
}

// GenerateWithPrefix builds "prefix-suffix" for the automatic slug mode.
func GenerateWithPrefix(prefix string, length int) (string, error) {
	suffix, err := GenerateSlug(length)
	if err != nil {
		return "", err
	}
	slug := prefix + "-" + suffix
	if len(slug) > maxSlugLen {
		return "", fmt.Errorf("prefix and slug length exceed %d characters", maxSlugLen)
	}
	return slug, nil
}
