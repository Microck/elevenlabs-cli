package elevenlabs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Microck/elevenlabs-web-cli/internal/testengine"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestNoRedirectOrAutomaticRetry(t *testing.T) {
	for _, status := range []int{302, 401, 403, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Location", "/unexpected")
				w.WriteHeader(status)
			}))
			defer server.Close()
			client := New(&Session{GalletonID: "fixture", UserAgent: "Mozilla/5.0 fixture"})
			client.BaseURL = server.URL
			client.Engine = testengine.Fixture(t, Adapter("fixture-key"), server.URL).Client
			_, err := client.JSON(context.Background(), "POST", "/v1/test", map[string]string{"text": "hello"})
			if err == nil || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
			expected := map[int]int{302: 6, 401: 3, 403: 4, 429: 5, 500: 6}[status]
			var apiError *Error
			if !errors.As(err, &apiError) || apiError.Code != expected {
				t.Fatalf("wrong error: %v", err)
			}
		})
	}
}

func TestSessionFilesAndValidation(t *testing.T) {
	dir := t.TempDir()
	path, err := SessionPath(dir, "example")
	if err != nil {
		t.Fatal(err)
	}
	original := &Session{UserAgent: "Mozilla/5.0 fixture", APIKey: "fixture-key", GalletonID: "fixture"}
	if err := SaveSession(path, original); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	if err := SaveSession(path, &Session{}); err == nil {
		t.Fatal("accepted empty session")
	}
	saved, err := LoadSession(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, original) {
		t.Fatal("invalid import overwrote good session")
	}
	for _, name := range []string{"../escape", "", "a/b", "a\\b", ".hidden"} {
		if _, err := SessionPath(dir, name); err == nil {
			t.Errorf("accepted profile %q", name)
		}
	}
	profiles, err := Profiles(dir)
	if err != nil || strings.Join(profiles, ",") != "example" {
		t.Fatalf("%v %v", profiles, err)
	}
}

func TestPreserveLargeJSONNumbers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"id":9007199254740993}`) }))
	defer server.Close()
	client := New(&Session{GalletonID: "fixture", UserAgent: "Mozilla/5.0 fixture"})
	client.BaseURL = server.URL
	client.Engine = testengine.Fixture(t, Adapter("fixture-key"), server.URL).Client
	value, err := client.JSON(context.Background(), "GET", "/v1/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(value)
	if string(raw) != `{"id":9007199254740993}` {
		t.Fatal(string(raw))
	}
}

func TestGenerationRestrictionIsNotExpiredSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"detail":{"status":"detected_unusual_activity","message":"Unusual activity has been detected on your account, so Free Tier access has been disabled."}}`)
	}))
	defer server.Close()
	client := New(&Session{GalletonID: "fixture", UserAgent: "Mozilla/5.0 fixture"})
	client.BaseURL = server.URL
	client.Engine = testengine.Fixture(t, Adapter("fixture-key"), server.URL).Client
	_, err := client.JSON(context.Background(), "POST", "/v1/text-to-dialogue/stream", nil)
	var restriction *Error
	if !errors.As(err, &restriction) || restriction.Code != 4 || restriction.Status != 401 {
		t.Fatalf("wrong restriction: %v", err)
	}
	if strings.Contains(err.Error(), "login") {
		t.Fatal("restriction must not suggest another login")
	}
}
