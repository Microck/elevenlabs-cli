package elevenlabs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

// BrowserSession is read only during import; Galleton owns durable credentials.
type BrowserSession struct {
	UserAgent    string    `json:"user_agent"`
	APIKey       string    `json:"api_key"`
	RefreshToken string    `json:"refresh_token"`
	AccessToken  string    `json:"access_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (s BrowserSession) Validate() error {
	if s.UserAgent == "" || s.APIKey == "" || s.RefreshToken == "" || s.AccessToken == "" || s.ExpiresAt.IsZero() {
		return &Error{Code: 2, Message: "session needs user_agent, api_key, refresh_token, access_token and expires_at; export a fresh browser session"}
	}
	return nil
}

// Session is a non-secret reference to credentials in the Galleton vault.
type Session struct {
	GalletonID string   `json:"galleton_id"`
	RetiredIDs []string `json:"retired_ids,omitempty"`
	APIKey     string   `json:"api_key"`
	UserAgent  string   `json:"user_agent"`
}

func (s Session) Validate() error {
	if s.GalletonID == "" || s.APIKey == "" || s.UserAgent == "" {
		return &Error{Code: 3, Message: "profile needs a Galleton session; run auth login with a fresh browser export"}
	}
	return nil
}

var profileName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

func SessionPath(dir, profile string) (string, error) {
	if !profileName.MatchString(profile) {
		return "", &Error{Code: 2, Message: "profile must contain 1-64 letters, numbers, underscores or hyphens and start with a letter or number"}
	}
	return filepath.Join(dir, "profiles", profile+".json"), nil
}

func LoadSession(path string) (*Session, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, &Error{Code: 3, Message: "no session for this profile; run elevenlabs auth login --session FILE"}
	}
	if err != nil {
		return nil, err
	}
	var session Session
	if err := json.Unmarshal(raw, &session); err != nil {
		return nil, fmt.Errorf("invalid session file: %w", err)
	}
	if err := session.Validate(); err != nil {
		return nil, err
	}
	return &session, nil
}

// SaveSession replaces a profile reference only after the complete file is synced.
func SaveSession(path string, session *Session) error {
	if err := session.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return WritePrivateFile(path, raw)
}

// WritePrivateFile atomically replaces a private file after writing and syncing it.
func WritePrivateFile(path string, raw []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".session-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(raw); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func Profiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(dir, "profiles"))
	names := []string{}
	if os.IsNotExist(err) {
		return names, nil
	}
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && filepath.Ext(name) == ".json" {
			profile := name[:len(name)-5]
			if profileName.MatchString(profile) {
				names = append(names, profile)
			}
		}
	}
	sort.Strings(names)
	return names, nil
}
