package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	envFile := "DOTENV_TEST_KEY=hello\nDOTENV_TEST_QUOTED=\"quoted value\"\n# comment line\nBAD_LINE_WITHOUT_EQUALS\nDOTENV_TEST_EMPTY=\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envFile), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTENV_TEST_PRESET", "preset")

	loadDotEnv()

	if got := os.Getenv("DOTENV_TEST_KEY"); got != "hello" {
		t.Fatalf("DOTENV_TEST_KEY = %q", got)
	}
	if got := os.Getenv("DOTENV_TEST_QUOTED"); got != "quoted value" {
		t.Fatalf("DOTENV_TEST_QUOTED = %q", got)
	}
	if got := os.Getenv("DOTENV_TEST_EMPTY"); got != "" {
		t.Fatalf("DOTENV_TEST_EMPTY = %q", got)
	}
}

func TestLoadDotEnvDoesNotOverrideExistingEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("DOTENV_TEST_OVERRIDE", "from-environment")

	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DOTENV_TEST_OVERRIDE=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	loadDotEnv()

	if got := os.Getenv("DOTENV_TEST_OVERRIDE"); got != "from-environment" {
		t.Fatalf("env file must not override real env, got %q", got)
	}
}

func TestLoadDotEnvMissingFileIsFine(t *testing.T) {
	t.Chdir(t.TempDir())
	loadDotEnv()
}

func TestParseDotEnvValue(t *testing.T) {
	cases := map[string]string{
		"true                       # a comment": "true",
		"http://localhost:8080 # origin":         "http://localhost:8080",
		"no#hash":                                "no#hash",
		"\"quoted # inside\" # comment":          "quoted # inside",
		"'single value'":                         "single value",
		"# leading comment":                      "",
		"plain":                                  "plain",
		"":                                       "",
	}
	for in, want := range cases {
		if got := parseDotEnvValue(in); got != want {
			t.Fatalf("parseDotEnvValue(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLoadDotEnvInlineComments(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, ".env"),
		[]byte("DOTENV_TEST_PORT=8090   # api port\nDOTENV_TEST_ORIGINS=http://localhost:3000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loadDotEnv()
	if got := os.Getenv("DOTENV_TEST_PORT"); got != "8090" {
		t.Fatalf("DOTENV_TEST_PORT = %q, want 8090", got)
	}
	if got := os.Getenv("DOTENV_TEST_ORIGINS"); got != "http://localhost:3000" {
		t.Fatalf("DOTENV_TEST_ORIGINS = %q", got)
	}
}
