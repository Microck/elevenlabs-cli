package elevenlabs

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Microck/elevenlabs-web-cli/internal/galleton"
	"github.com/Microck/elevenlabs-web-cli/internal/testengine"
)

func TestManagedRotationSurvivesRestart(t *testing.T) {
	var mu sync.Mutex
	refresh, access := "first-refresh", "first-access"
	renewals := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Header.Get("User-Agent") != "Mozilla/5.0 fixture" {
			t.Error("browser user agent changed")
		}
		if r.URL.Path == "/token" {
			_ = r.ParseForm()
			if r.Method != "POST" || r.PostForm.Get("refresh_token") != refresh || r.PostForm.Get("grant_type") != "refresh_token" || r.URL.Query().Get("key") != "fixture-key" || r.Header.Get("Referer") != "https://elevenlabs.io/" {
				w.WriteHeader(401)
				return
			}
			renewals++
			refresh = fmt.Sprintf("rotated-refresh-%d", renewals)
			access = fmt.Sprintf("rotated-access-%d", renewals)
			fmt.Fprintf(w, `{"access_token":%q,"refresh_token":%q,"expires_in":"3600"}`, access, refresh)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+access {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path == "/v1/fail" {
			w.WriteHeader(503)
			return
		}
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()
	adapter := testengine.Config(Adapter("fixture-key"), server.URL, server.URL)
	engine := testengine.Start(t, adapter)
	expired := time.Now().Add(-time.Hour)
	ctx := context.Background()
	if _, err := engine.Client.Connect(ctx, "account", galleton.Credentials{Provider: "elevenlabs", AccessToken: access, RefreshToken: refresh, AccessExpiresAt: &expired, Secrets: map[string]string{"user_agent": "Mozilla/5.0 fixture"}}); err != nil {
		t.Fatal(err)
	}
	session := &Session{GalletonID: "account", APIKey: "fixture-key", UserAgent: "Mozilla/5.0 fixture"}
	client := New(session)
	client.Engine = engine.Client
	client.BaseURL = server.URL
	if _, err := client.JSON(ctx, "GET", "/v1/fail", nil); err == nil {
		t.Fatal("expected API failure")
	}
	engine.Stop()
	files, _ := filepath.Glob(filepath.Join(engine.Dir, "*.session-v2"))
	if len(files) != 1 {
		t.Fatal("missing encrypted session")
	}
	raw, err := os.ReadFile(files[0])
	if err != nil || strings.Contains(string(raw), "rotated-refresh") {
		t.Fatal("vault contains plaintext or unreadable")
	}
	engine = testengine.Restart(t, engine.Dir, adapter)
	client.Engine = engine.Client
	if err := client.Refresh(ctx); err != nil {
		t.Fatal("lost rotated credential:", err)
	}
	if _, err := client.JSON(ctx, "GET", "/v1/models", nil); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	count := renewals
	mu.Unlock()
	if count != 2 {
		t.Fatalf("expected two serialized renewals, got %d", count)
	}
	if _, err := engine.Client.Headers(ctx, "account", "https://example.com"); err == nil {
		t.Fatal("engine leaked bearer outside allowed origin")
	}
	if err := engine.Client.Forget(ctx, "account"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.JSON(ctx, "GET", "/v1/models", nil); err == nil {
		t.Fatal("forgotten session still usable")
	}
}
