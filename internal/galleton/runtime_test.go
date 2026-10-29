package galleton

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Microck/elevenlabs-cli/internal/galleton/bundle"
)

func TestMain(m *testing.M) {
	if handled, err := HandleHost(os.Args[1:]); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestManagedHostLeasesAndRestart(t *testing.T) {
	if len(bundle.Binary) > 0 {
		t.Setenv("PATH", t.TempDir())
	}
	for _, key := range []string{"ELEVENLABS_GALLETON_DIR", "ELEVENLABS_GALLETON_URL", "ELEVENLABS_GALLETON_TOKEN_FILE"} {
		t.Setenv(key, "")
	}
	dir := filepath.Join(t.TempDir(), "galleton")
	adapter := []byte(`{"providers":[{"name":"elevenlabs","kind":"http","origins":["http://127.0.0.1:1"],"allow_http_loopback":true,"refresh_url":"http://127.0.0.1:1/token","body_format":"form","body":{"refresh_token":"${refresh_token}"},"response":{"access_token_pointer":"/access_token"},"refresh_interval_seconds":3600}]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	setup := Setup{Adapter: adapter, Launched: func(c *Client) error { return c.Resume(ctx, "elevenlabs") }}
	first, err := Open(ctx, dir, setup)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	t.Cleanup(func() {
		if err := Stop(context.Background(), dir); err != nil {
			t.Error(err)
		}
	})
	second, err := Open(ctx, dir, setup)
	if err != nil {
		t.Fatal(err)
	}
	if err := Stop(ctx, dir); err == nil {
		t.Fatal("stopped leased engine")
	}
	second.Close()
	first.Close()
	if err := Stop(ctx, dir); err != nil {
		t.Fatal(err)
	}
	if RuntimeStatus(ctx, dir)["running"] != false {
		t.Fatal("engine still running")
	}
	again, err := Open(ctx, dir, setup)
	if err != nil {
		t.Fatal(err)
	}
	if err := again.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	again.Close()
}
