// Command cutsh is a terminal client for the cut.sh API.
//
// Configuration precedence: flags > env (CUTSH_API_URL, CUTSH_TOKEN) >
// ~/.config/cutsh/config.json.
//
//	cutsh login [--api URL] [--email E --password P]
//	cutsh logout
//	cutsh whoami
//	cutsh add <url> [--slug S] [--prefix P] [--title T]
//	cutsh list [--limit N] [--q QUERY]
//	cutsh delete <id>
//	cutsh stats <id> [--days N]
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultAPI = "http://localhost:8080"

type config struct {
	APIURL string `json:"api_url"`
	Token  string `json:"token"`
}

func configPath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "cutsh", "config.json")
	}
	return filepath.Join(".", ".cutsh.json")
}

func loadConfig() config {
	var c config
	if data, err := os.ReadFile(configPath()); err == nil {
		_ = json.Unmarshal(data, &c)
	}
	if v := os.Getenv("CUTSH_API_URL"); v != "" {
		c.APIURL = v
	}
	if v := os.Getenv("CUTSH_TOKEN"); v != "" {
		c.Token = v
	}
	if c.APIURL == "" {
		c.APIURL = defaultAPI
	}
	return c
}

func saveConfig(c config) error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func request(method, base, path, token string, body any) ([]byte, int, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, strings.TrimRight(base, "/")+path, rd)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return data, resp.StatusCode, nil
}

func fail(body []byte, status int) {
	var e apiError
	if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
		fatalf("error (%s): %s", e.Error.Code, e.Error.Message)
	}
	fatalf("request failed with status %d", status)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "cutsh: "+format+"\n", args...)
	os.Exit(1)
}

func readPassword(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	var pw string
	_, _ = fmt.Fscanln(os.Stdin, &pw)
	return strings.TrimSpace(pw)
}

// valueFlags lists flags that consume the following argument, so positionals
// can be moved after flags before flag.Parse (which stops at the first
// non-flag argument).
var valueFlags = map[string]bool{
	"-api": true, "--api": true,
	"-email": true, "--email": true,
	"-password": true, "--password": true,
	"-slug": true, "--slug": true,
	"-prefix": true, "--prefix": true,
	"-title": true, "--title": true,
	"-limit": true, "--limit": true,
	"-q": true, "--q": true,
	"-days": true, "--days": true,
}

// reorderArgs moves positional arguments after flags so `cutsh add <url>
// --slug x` and `cutsh add --slug x <url>` both work.
func reorderArgs(args []string) []string {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if valueFlags[a] && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positionals = append(positionals, a)
		}
	}
	return append(flags, positionals...)
}

func cmdLogin(args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	api := fs.String("api", "", "API base URL")
	email := fs.String("email", "", "account email")
	password := fs.String("password", "", "account password (prompted when empty)")
	_ = fs.Parse(reorderArgs(args))

	c := loadConfig()
	if *api != "" {
		c.APIURL = *api
	}
	if *email == "" {
		fmt.Fprint(os.Stderr, "email: ")
		var e string
		_, _ = fmt.Fscanln(os.Stdin, &e)
		*email = strings.TrimSpace(e)
	}
	if *password == "" {
		*password = readPassword("password: ")
	}
	body, status, err := request(http.MethodPost, c.APIURL, "/api/v1/auth/login", "", map[string]string{
		"email": *email, "password": *password,
	})
	if err != nil {
		fatalf("%v", err)
	}
	if status != http.StatusOK {
		fail(body, status)
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || resp.Token == "" {
		fatalf("unexpected login response")
	}
	c.Token = resp.Token
	if err := saveConfig(c); err != nil {
		fatalf("save config: %v", err)
	}
	fmt.Printf("logged in; token stored in %s\n", configPath())
}

func cmdLogout(args []string) {
	c := loadConfig()
	if c.Token != "" {
		_, _, _ = request(http.MethodPost, c.APIURL, "/api/v1/auth/logout", c.Token, nil)
	}
	c.Token = ""
	if err := saveConfig(c); err != nil {
		fatalf("save config: %v", err)
	}
	fmt.Println("logged out")
}

func cmdWhoami(args []string) {
	c := loadConfig()
	body, status, err := request(http.MethodGet, c.APIURL, "/api/v1/me", c.Token, nil)
	if err != nil {
		fatalf("%v", err)
	}
	if status != http.StatusOK {
		fail(body, status)
	}
	var resp struct {
		User struct {
			Email string `json:"email"`
			ID    int64  `json:"id"`
		} `json:"user"`
	}
	_ = json.Unmarshal(body, &resp)
	fmt.Printf("user %d <%s> via %s\n", resp.User.ID, resp.User.Email, c.APIURL)
}

type urlItem struct {
	ID          int64   `json:"id"`
	Slug        string  `json:"slug"`
	ShortURL    string  `json:"short_url"`
	OriginalURL string  `json:"original_url"`
	Title       *string `json:"title"`
	ClickCount  int64   `json:"click_count"`
	CreatedAt   string  `json:"created_at"`
}

func cmdAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	slug := fs.String("slug", "", "exact custom slug")
	prefix := fs.String("prefix", "", "prefix for a generated slug")
	title := fs.String("title", "", "optional title")
	_ = fs.Parse(reorderArgs(args))
	if fs.NArg() != 1 {
		fatalf("usage: cutsh add <url> [--slug S] [--prefix P] [--title T]")
	}
	c := loadConfig()
	payload := map[string]string{"original_url": fs.Arg(0)}
	if *slug != "" {
		payload["custom_slug"] = *slug
	}
	if *prefix != "" {
		payload["prefix"] = *prefix
	}
	if *title != "" {
		payload["title"] = *title
	}
	body, status, err := request(http.MethodPost, c.APIURL, "/api/v1/urls", c.Token, payload)
	if err != nil {
		fatalf("%v", err)
	}
	if status != http.StatusCreated {
		fail(body, status)
	}
	var item urlItem
	if err := json.Unmarshal(body, &item); err != nil {
		fatalf("unexpected response: %v", err)
	}
	fmt.Println(item.ShortURL)
}

func cmdList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	limit := fs.Int("limit", 20, "max results")
	q := fs.String("q", "", "search query")
	_ = fs.Parse(reorderArgs(args))
	c := loadConfig()
	path := fmt.Sprintf("/api/v1/urls?limit=%d", *limit)
	if *q != "" {
		path += "&q=" + *q
	}
	body, status, err := request(http.MethodGet, c.APIURL, path, c.Token, nil)
	if err != nil {
		fatalf("%v", err)
	}
	if status != http.StatusOK {
		fail(body, status)
	}
	var resp struct {
		Data []urlItem `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		fatalf("unexpected response: %v", err)
	}
	if len(resp.Data) == 0 {
		fmt.Println("no links")
		return
	}
	for _, it := range resp.Data {
		title := ""
		if it.Title != nil {
			title = " — " + *it.Title
		}
		fmt.Printf("%6d  %s  (%d clicks)%s\n", it.ID, it.ShortURL, it.ClickCount, title)
	}
}

func cmdDelete(args []string) {
	if len(args) != 1 {
		fatalf("usage: cutsh delete <id>")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fatalf("invalid id %q", args[0])
	}
	c := loadConfig()
	body, status, err := request(http.MethodDelete, c.APIURL, fmt.Sprintf("/api/v1/urls/%d", id), c.Token, nil)
	if err != nil {
		fatalf("%v", err)
	}
	if status != http.StatusNoContent {
		fail(body, status)
	}
	fmt.Printf("deleted %d\n", id)
}

func cmdStats(args []string) {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	days := fs.Int("days", 7, "series window in days")
	_ = fs.Parse(reorderArgs(args))
	if fs.NArg() != 1 {
		fatalf("usage: cutsh stats <id> [--days N]")
	}
	id, err := strconv.ParseInt(fs.Arg(0), 10, 64)
	if err != nil {
		fatalf("invalid id %q", fs.Arg(0))
	}
	c := loadConfig()
	body, status, err := request(http.MethodGet, c.APIURL,
		fmt.Sprintf("/api/v1/urls/%d/analytics?days=%d", id, *days), c.Token, nil)
	if err != nil {
		fatalf("%v", err)
	}
	if status != http.StatusOK {
		fail(body, status)
	}
	var s struct {
		TotalClicks  int64 `json:"total_clicks"`
		Clicks24h    int64 `json:"clicks_24h"`
		TopReferrers []struct {
			Label string `json:"label"`
			Count int64  `json:"count"`
		} `json:"top_referrers"`
		Devices []struct {
			Label string `json:"label"`
			Count int64  `json:"count"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(body, &s); err != nil {
		fatalf("unexpected response: %v", err)
	}
	fmt.Printf("total: %d clicks · last 24h: %d\n", s.TotalClicks, s.Clicks24h)
	if len(s.TopReferrers) > 0 {
		fmt.Print("top referrers:")
		for _, r := range s.TopReferrers {
			fmt.Printf(" %s(%d)", r.Label, r.Count)
		}
		fmt.Println()
	}
	if len(s.Devices) > 0 {
		fmt.Print("devices:")
		for _, d := range s.Devices {
			fmt.Printf(" %s(%d)", d.Label, d.Count)
		}
		fmt.Println()
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `cutsh — cut.sh terminal client

commands:
  login [--api URL] [--email E]   authenticate and store a session token
  logout                          revoke the stored token
  whoami                          show the authenticated user
  add <url> [--slug S] [--prefix P] [--title T]
  list [--limit N] [--q QUERY]
  delete <id>
  stats <id> [--days N]

env: CUTSH_API_URL, CUTSH_TOKEN (API keys work for add/list/stats/delete)
config: `+configPath()+`
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "login":
		cmdLogin(args)
	case "logout":
		cmdLogout(args)
	case "whoami":
		cmdWhoami(args)
	case "add":
		cmdAdd(args)
	case "list", "ls":
		cmdList(args)
	case "delete", "rm":
		cmdDelete(args)
	case "stats":
		cmdStats(args)
	case "help", "-h", "--help":
		usage()
	default:
		fatalf("unknown command %q (try cutsh help)", cmd)
	}
}
