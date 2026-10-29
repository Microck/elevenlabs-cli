package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Microck/elevenlabs-cli/internal/elevenlabs"
	"github.com/Microck/elevenlabs-cli/internal/galleton"
	"github.com/Microck/elevenlabs-cli/internal/testengine"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(args []string, input string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), "test", args, strings.NewReader(input), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestManagedProfileWorkflow(t *testing.T) {
	run := managedRunner(t)
	dir := t.TempDir()
	t.Setenv("ELEVENLABS_CONFIG_DIR", dir)
	session := `{"user_agent":"Mozilla/5.0 fixture","api_key":"fixture-key","access_token":"fixture-access","refresh_token":"fixture-refresh","expires_at":"2099-01-01T00:00:00Z"}`
	code, out, err := run([]string{"auth", "login", "--session", "-", "--profile", "test"}, session)
	if code != 0 {
		t.Fatalf("%d %s", code, err)
	}
	if strings.Contains(out, "fixture-") {
		t.Fatal("login leaked credentials")
	}
	code, out, err = run([]string{"auth", "status", "--profile", "test"}, "")
	if code != 0 || !json.Valid([]byte(out)) || err != "" {
		t.Fatalf("%d %s %s", code, out, err)
	}
	if strings.Contains(out, "fixture-") {
		t.Fatal("status leaked credentials")
	}
	code, out, err = run([]string{"profile", "list", "--format", "jsonl"}, "")
	if code != 0 || out != "\"test\"\n" {
		t.Fatalf("%d %s %s", code, out, err)
	}
	code, _, _ = run([]string{"auth", "logout", "--profile", "test", "--no-input"}, "")
	if code != 2 {
		t.Fatalf("unconfirmed logout: %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "profiles", "test.json")); err != nil {
		t.Fatal(err)
	}
	code, _, err = run([]string{"auth", "logout", "--profile", "test", "--yes"}, "")
	if code != 0 {
		t.Fatal(err)
	}
	code, _, _ = run([]string{"auth", "status", "--profile", "test"}, "")
	if code != 3 {
		t.Fatalf("missing auth code %d", code)
	}
}

func TestOfflineHelpAndErrors(t *testing.T) {
	t.Setenv("ELEVENLABS_CONFIG_DIR", filepath.Join(t.TempDir(), "missing"))
	for _, args := range [][]string{{"--help"}, {"auth", "login", "--help"}, {"--version"}} {
		code, out, err := run(args, "")
		if code != 0 || out == "" || err != "" {
			t.Fatalf("%v: %d %s %s", args, code, out, err)
		}
	}
	for _, args := range [][]string{{"model", "list", "--format", "wrong"}, {"model", "list", "extra"}, {"auth", "login", "--profile", "../bad", "--session", "-"}, {"--bad-flag"}} {
		code, out, err := run(args, "")
		if code != 2 || out != "" || err == "" {
			t.Fatalf("%v: %d %s %s", args, code, out, err)
		}
	}
	code, out, err := run([]string{"me", "subscription", "--error-format", "json"}, "")
	if code != 3 || out != "" || !json.Valid([]byte(err)) {
		t.Fatalf("%d %s %s", code, out, err)
	}
}

func TestBrowsingRejectsInvalidInputsBeforeAuth(t *testing.T) {
	t.Setenv("ELEVENLABS_CONFIG_DIR", t.TempDir())
	for _, args := range [][]string{
		{"voice", "search", ""},
		{"voice", "search", "narrator", "--page", "-1"},
		{"voice", "search", "narrator", "--page-size", "0"},
		{"history", "list", "--page-size", "101"},
	} {
		code, out, stderr := run(args, "")
		if code != 2 || out != "" || stderr == "" {
			t.Fatalf("%v: %d %q %q", args, code, out, stderr)
		}
	}
}

func TestUnknownCommandUsesUsageExitAndJSONError(t *testing.T) {
	for _, args := range [][]string{{"nope", "--error-format", "json"}, {"voice", "nope", "--error-format", "json"}} {
		code, out, stderr := run(args, "")
		if code != 2 || out != "" || !json.Valid([]byte(stderr)) {
			t.Fatalf("%v: %d %q %q", args, code, out, stderr)
		}
	}
}

func TestDefaultProfileSelectionAndExplicitOverride(t *testing.T) {
	run := managedRunner(t)
	t.Setenv("ELEVENLABS_CONFIG_DIR", t.TempDir())
	session := `{"user_agent":"Mozilla/5.0 fixture","api_key":"fixture-key","access_token":"fixture-access","refresh_token":"fixture-refresh","expires_at":"2099-01-01T00:00:00Z"}`
	for _, name := range []string{"default", "work"} {
		if code, _, err := run([]string{"auth", "login", "--profile", name, "--session", "-"}, session); code != 0 {
			t.Fatal(err)
		}
	}
	if code, _, err := run([]string{"profile", "use", "work"}, ""); code != 0 {
		t.Fatal(err)
	}
	for _, check := range []struct {
		args    []string
		profile string
	}{
		{[]string{"auth", "status"}, "work"},
		{[]string{"auth", "status", "--profile", "default"}, "default"},
	} {
		code, out, err := run(check.args, "")
		var status map[string]any
		if code != 0 || json.Unmarshal([]byte(out), &status) != nil || status["profile"] != check.profile {
			t.Fatalf("%v: %d %s %s", check.args, code, out, err)
		}
	}
	if code, _, _ := run([]string{"profile", "remove", "work"}, ""); code != 2 {
		t.Fatal("unconfirmed removal succeeded")
	}
	if code, _, err := run([]string{"profile", "remove", "work", "--yes"}, ""); code != 0 {
		t.Fatal(err)
	}
	if code, _, _ := run([]string{"auth", "status"}, ""); code != 3 {
		t.Fatalf("removed active profile should require login: %d", code)
	}
	if code, out, err := run([]string{"config", "get", "default_profile"}, ""); code != 0 || out != "\"work\"\n" {
		t.Fatalf("selection changed: %d %s %s", code, out, err)
	}
	if code, _, err := run([]string{"config", "set", "default_profile", "default"}, ""); code != 0 {
		t.Fatal(err)
	}
}

func TestConfigFailureAndOfflineGuides(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEVENLABS_CONFIG_DIR", dir)
	for _, args := range [][]string{{"auth", "export-script"}, {"skills", "list"}, {"skills", "get", "elevenlabs"}, {"completion", "bash"}, {"completion", "zsh"}, {"completion", "fish"}, {"completion", "powershell"}} {
		code, out, err := run(args, "")
		if code != 0 || out == "" || err != "" {
			t.Fatalf("%v: %d %q %s", args, code, out, err)
		}
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := run([]string{"config", "list"}, ""); code == 0 {
		t.Fatal("accepted malformed config")
	}
	if code, out, err := run([]string{"config", "path"}, ""); code != 0 || !strings.Contains(out, "config.json") {
		t.Fatalf("recovery path: %d %s %s", code, out, err)
	}
	if raw, err := os.ReadFile(path); err != nil || string(raw) != "broken" {
		t.Fatal("malformed config overwritten")
	}
}

func managedRunner(t *testing.T) func([]string, string) (int, string, string) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_ = r.ParseForm()
			if r.PostForm.Get("refresh_token") == "rejected-refresh" {
				w.WriteHeader(401)
				return
			}
			fmt.Fprint(w, `{"access_token":"fixture-access","refresh_token":"fixture-refresh","expires_in":"3600"}`)
			return
		}
		if r.Header.Get("Authorization") != "Bearer fixture-access" {
			w.WriteHeader(401)
			return
		}
		fmt.Fprint(w, `{"user_id":"fixture-account"}`)
	}))
	t.Cleanup(server.Close)
	engine := testengine.Start(t, testengine.Config(elevenlabs.Adapter("fixture-key"), server.URL, server.URL))
	t.Setenv("ELEVENLABS_GALLETON_URL", engine.URL)
	t.Setenv("ELEVENLABS_GALLETON_TOKEN_FILE", filepath.Join(engine.Dir, "api.token"))
	t.Setenv("ELEVENLABS_GALLETON_DIR", "")
	return func(args []string, input string) (int, string, string) {
		var out, stderr bytes.Buffer
		a := &app{stdin: strings.NewReader(input), stdout: &out, stderr: &stderr, apiBase: server.URL}
		code := a.execute(context.Background(), "test", args)
		return code, out.String(), stderr.String()
	}
}

func TestFailedReplacementPreservesManagedSession(t *testing.T) {
	run := managedRunner(t)
	dir := t.TempDir()
	t.Setenv("ELEVENLABS_CONFIG_DIR", dir)
	session := `{"user_agent":"Mozilla/5.0 fixture","api_key":"fixture-key","access_token":"fixture-access","refresh_token":"fixture-refresh","expires_at":"2099-01-01T00:00:00Z"}`
	if code, _, stderr := run([]string{"auth", "login", "--session", "-"}, session); code != 0 {
		t.Fatal(stderr)
	}
	path := filepath.Join(dir, "profiles", "default.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"fixture-access", "fixture-refresh", "access_token", "refresh_token"} {
		if strings.Contains(string(before), secret) {
			t.Fatal("profile contains token")
		}
	}
	rejected := strings.ReplaceAll(session, "fixture-refresh", "rejected-refresh")
	if code, _, _ := run([]string{"auth", "login", "--session", "-"}, rejected); code != 3 {
		t.Fatalf("rejected login exit %d", code)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("failed login replaced reference")
	}
	if code, _, stderr := run([]string{"auth", "status", "--check"}, ""); code != 0 {
		t.Fatal("prior session lost:", stderr)
	}
	var prior elevenlabs.Session
	if err := json.Unmarshal(before, &prior); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := run([]string{"auth", "login", "--session", "-"}, session); code != 0 {
		t.Fatal(stderr)
	}
	engine, err := galleton.FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = engine.Status(context.Background(), prior.GalletonID); !galleton.IsStatus(err, 404) {
		t.Fatal("retired login still present:", err)
	}
}
