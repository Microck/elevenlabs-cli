package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRenewBeforeRequestAndPersistOnFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles", "test.json")
	session := &Session{UserAgent: "Mozilla/5.0 fixture", APIKey: "test-key", RefreshToken: "old-refresh", AccessToken: "expired", ExpiresAt: time.Now().Add(-time.Hour)}
	if err := SaveSession(path, session); err != nil {
		t.Fatal(err)
	}
	var events []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "Mozilla/5.0 fixture" {
			t.Error("browser user agent not preserved")
		}
		switch r.URL.Path {
		case "/token":
			events = append(events, "refresh")
			if r.Header.Get("Referer") != "https://elevenlabs.io/" {
				t.Error("Firebase requires the app referrer")
			}
			if r.Method != "POST" || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
				t.Error("wrong token request")
			}
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if r.PostForm.Get("grant_type") != "refresh_token" || r.PostForm.Get("refresh_token") != "old-refresh" || r.URL.Query().Get("key") != "test-key" {
				t.Error("wrong token form")
			}
			fmt.Fprint(w, `{"access_token":"renewed-access","refresh_token":"renewed-refresh","expires_in":"3600"}`)
		case "/v1/models":
			events = append(events, "api")
			saved, err := LoadSession(path)
			if err != nil || saved.AccessToken != "renewed-access" {
				t.Error("API called before tokens persisted")
			}
			if r.Header.Get("Authorization") != "Bearer renewed-access" {
				t.Error("wrong bearer")
			}
			w.WriteHeader(503)
			fmt.Fprint(w, `{"secret":"must-not-appear"}`)
		default:
			t.Error("unexpected route")
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	client := New(session)
	client.BaseURL, client.TokenURL = server.URL, server.URL+"/token"
	client.Save = func(s *Session) error { return SaveSession(path, s) }
	var debug bytes.Buffer
	client.Debug = &debug
	_, err := client.JSON(context.Background(), "GET", "/v1/models", nil)
	var apiError *Error
	if !errors.As(err, &apiError) || apiError.Code != 6 {
		t.Fatalf("expected API failure: %v", err)
	}
	if strings.Join(events, ",") != "refresh,api" {
		t.Fatalf("unexpected calls: %v", events)
	}
	for _, secret := range []string{"renewed-access", "old-refresh", "test-key", "must-not-appear"} {
		if strings.Contains(debug.String()+err.Error(), secret) {
			t.Errorf("leaked %s", secret)
		}
	}
}

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
			client := New(&Session{AccessToken: "test", ExpiresAt: time.Now().Add(time.Hour)})
			client.BaseURL = server.URL
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

func TestSaveFailureStopsAPIRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			t.Error("API call after save failure")
		}
		fmt.Fprint(w, `{"access_token":"new","refresh_token":"new-refresh","expires_in":"3600"}`)
	}))
	defer server.Close()
	client := New(&Session{UserAgent: "Mozilla/5.0 fixture", APIKey: "key", RefreshToken: "old", AccessToken: "old", ExpiresAt: time.Now().Add(-time.Hour)})
	client.BaseURL, client.TokenURL = server.URL, server.URL+"/token"
	client.Save = func(*Session) error { return errors.New("disk full") }
	_, err := client.JSON(context.Background(), "GET", "/v1/models", nil)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("%v", err)
	}
	if client.Session.RefreshToken != "old" {
		t.Fatal("changed token despite failed save")
	}
}

func TestSessionFilesAndValidation(t *testing.T) {
	dir := t.TempDir()
	path, err := SessionPath(dir, "example")
	if err != nil {
		t.Fatal(err)
	}
	original := &Session{UserAgent: "Mozilla/5.0 fixture", APIKey: "key", AccessToken: "access", RefreshToken: "refresh", ExpiresAt: time.Now().UTC().Truncate(time.Second)}
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
	if *saved != *original {
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
	client := New(&Session{AccessToken: "test", ExpiresAt: time.Now().Add(time.Hour)})
	client.BaseURL = server.URL
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
	client := New(&Session{AccessToken: "test", ExpiresAt: time.Now().Add(time.Hour)})
	client.BaseURL = server.URL
	_, err := client.JSON(context.Background(), "POST", "/v1/text-to-dialogue/stream", nil)
	var restriction *Error
	if !errors.As(err, &restriction) || restriction.Code != 4 || restriction.Status != 401 {
		t.Fatalf("wrong restriction: %v", err)
	}
	if strings.Contains(err.Error(), "login") {
		t.Fatal("restriction must not suggest another login")
	}
}
