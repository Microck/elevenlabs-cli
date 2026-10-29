// Package testengine runs the real pinned Galleton daemon with disposable state.
package testengine

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Microck/elevenlabs-web-cli/internal/galleton"
)

type Engine struct {
	Client   *galleton.Client
	URL, Dir string
	Stop     func()
}

// Start only permits loopback upstreams and always uses a new private vault.
func Start(t *testing.T, adapter []byte) *Engine {
	t.Helper()
	dir := t.TempDir()
	return Restart(t, dir, adapter)
}
func Restart(t *testing.T, dir string, adapter []byte) *Engine {
	t.Helper()
	var config struct {
		Providers []struct {
			Origins []string `json:"origins"`
		} `json:"providers"`
	}
	if json.Unmarshal(adapter, &config) != nil {
		t.Fatal("invalid fixture adapter")
	}
	for _, p := range config.Providers {
		for _, origin := range p.Origins {
			u, _ := url.Parse(origin)
			if u == nil || u.Hostname() != "127.0.0.1" {
				t.Fatal("test rejects live origin")
			}
		}
	}
	binary := os.Getenv("ELEVENLABS_TEST_ENGINE")
	if binary == "" {
		t.Fatal("run make test to build the pinned test engine")
	}
	path := filepath.Join(dir, "adapter.json")
	if err := os.WriteFile(path, adapter, 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(binary, "init", "--dir", dir).CombinedOutput(); err != nil {
		t.Fatalf("init: %v %s", err, output)
	}
	cmd := exec.Command(binary, "serve", "--dir", dir, "--config", path, "--listen", "127.0.0.1:0")
	pipe, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	address := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Galleton listening on ") {
				select {
				case address <- "http://" + strings.Fields(line)[3]:
				default:
				}
			}
		}
	}()
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	var base string
	select {
	case base = <-address:
	case err = <-done:
		t.Fatalf("daemon exited: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("daemon readiness timeout")
	}
	token, err := os.ReadFile(filepath.Join(dir, "api.token"))
	if err != nil {
		t.Fatal(err)
	}
	client, err := galleton.New(base, string(token))
	if err != nil {
		t.Fatal(err)
	}
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Shutdown(ctx); err != nil {
			t.Error(err)
		}
		select {
		case <-done:
		case <-ctx.Done():
			t.Error("daemon did not drain")
		}
	}
	t.Cleanup(stop)
	return &Engine{Client: client, URL: base, Dir: dir, Stop: stop}
}

func Config(raw []byte, tokenBase, apiBase string) []byte {
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		panic(err)
	}
	provider := config["providers"].([]any)[0].(map[string]any)
	provider["origins"] = []string{tokenBase, apiBase}
	if tokenBase == apiBase {
		provider["origins"] = []string{apiBase}
	}
	provider["refresh_url"] = tokenBase + "/token?key=fixture-key"
	provider["allow_http_loopback"] = true
	out, _ := json.Marshal(config)
	return out
}

// Fixture supplies an ordinary Firebase-shaped renewal response for API tests.
func Fixture(t *testing.T, adapter []byte, apiBase string) *Engine {
	t.Helper()
	token := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"access_token":"fixture","refresh_token":"fixture-refresh","expires_in":"3600"}`)
	}))
	t.Cleanup(token.Close)
	engine := Start(t, Config(adapter, token.URL, apiBase))
	expires := time.Now().Add(time.Hour)
	if _, err := engine.Client.Connect(context.Background(), "fixture", galleton.Credentials{Provider: "elevenlabs", AccessToken: "fixture", RefreshToken: "fixture-refresh", AccessExpiresAt: &expires, Secrets: map[string]string{"user_agent": "Mozilla/5.0 fixture"}}); err != nil {
		t.Fatal(err)
	}
	return engine
}
