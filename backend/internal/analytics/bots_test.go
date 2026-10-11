package analytics

import "testing"

func TestIsBot(t *testing.T) {
	bots := []string{
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
		"curl/8.4.0",
		"python-requests/2.31.0",
		"facebookexternalhit/1.1",
		"Twitterbot/1.0",
		"Slackbot-LinkExpanding 1.0",
		"Go-http-client/2.0",
		"Mozilla/5.0 (compatible; bingbot/2.0)",
		"Chrome-Lighthouse",
		"uptimerobot/2.0",
	}
	for _, ua := range bots {
		if !IsBot(ua) {
			t.Errorf("IsBot(%q) = false", ua)
		}
	}
	humans := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)",
		"Mozilla/5.0 (Linux; Android 14)",
		"",
	}
	for _, ua := range humans {
		if IsBot(ua) {
			t.Errorf("IsBot(%q) = true", ua)
		}
	}
}
