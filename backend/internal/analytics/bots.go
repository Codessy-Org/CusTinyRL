package analytics

import "strings"

// botTokens are matched (case-insensitive, substring) against the User-Agent
// to decide whether a click should be excluded from analytics. This is a
// privacy-conscious allowlist-style filter: raw UAs are never stored.
var botTokens = []string{
	// crawlers
	"bot", "crawler", "crawling", "spider", "slurp", "headless", "phantomjs",
	"bingpreview", "yandex", "duckduckbot", "baiduspider", "sogou", "exabot",
	"facebot", "archive.org_bot", "ia_archiver",
	// link previewers / social
	"facebookexternalhit", "twitterbot", "whatsapp", "telegrambot", "slackbot",
	"discordbot", "embedly", "pinterest", "linkedinbot", "skypeuripreview",
	"apple-podcast", "preview",
	// HTTP clients and scripts
	"curl/", "wget/", "python-requests", "python-httpx", "aiohttp", "httpie",
	"postman", "insomnia", "go-http-client", "java/", "okhttp", "libwww",
	"scrapy", "httpclient", "node-fetch", "axios/", "got (", "urllib",
	// uptime / monitoring / scraping
	"pingdom", "uptimer", "statuscake", "gtmetrix", "lighthouse",
	"chrome-lighthouse", "pagespeed", "monitor", "checkly", "newrelicpinger",
	"datadog", "zabbix", "nagios", "site24x7", "catchpoint",
	// validators / feed readers
	"validator", "w3.org", "feedfetcher", "feedparser", "rss", "syndication",
	"newsblur", "netnewswire", "liferea",
}

// tabletTokens / mobileTokens classify remaining human traffic coarsely.
var tabletTokens = []string{"ipad", "tablet", "kindle", "silk"}
var mobileTokens = []string{"mobi", "iphone", "android", "blackberry", "windows phone", "opera mini"}

// IsBot reports whether the user agent looks like a crawler, monitor,
// script, or link-preview fetcher rather than a human browser.
func IsBot(ua string) bool {
	if ua == "" {
		return false
	}
	l := strings.ToLower(ua)
	for _, t := range botTokens {
		if strings.Contains(l, t) {
			return true
		}
	}
	return false
}

// ClassifyDevice maps a user agent to a coarse device category.
func ClassifyDevice(ua string) string {
	if ua == "" {
		return ""
	}
	l := strings.ToLower(ua)
	for _, t := range botTokens {
		if strings.Contains(l, t) {
			return "bot"
		}
	}
	for _, t := range tabletTokens {
		if strings.Contains(l, t) {
			return "tablet"
		}
	}
	for _, t := range mobileTokens {
		if strings.Contains(l, t) {
			return "mobile"
		}
	}
	return "desktop"
}
