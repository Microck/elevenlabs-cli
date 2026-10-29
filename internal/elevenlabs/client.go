// Package elevenlabs implements the requests observed in the ElevenLabs web app.
package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const APIBase = "https://api.us.elevenlabs.io"
const TokenURL = "https://securetoken.googleapis.com/v1/token"

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"http_status,omitempty"`
}

func (e *Error) Error() string { return e.Message }

type Client struct {
	HTTP     *http.Client
	BaseURL  string
	TokenURL string
	Session  *Session
	Save     func(*Session) error
	Debug    io.Writer
}

func New(session *Session) *Client {
	return &Client{HTTP: &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}, BaseURL: APIBase, TokenURL: TokenURL, Session: session}
}

func (c *Client) send(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", c.Session.UserAgent)
	// Firebase restricts the web application's key to this referrer too.
	req.Header.Set("Origin", "https://elevenlabs.io")
	req.Header.Set("Referer", "https://elevenlabs.io/")
	started := time.Now()
	response, err := c.HTTP.Do(req)
	if c.Debug != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		fmt.Fprintf(c.Debug, "%s %s %d %s\n", req.Method, req.URL.EscapedPath(), status, time.Since(started).Round(time.Millisecond))
	}
	if err != nil {
		var urlError *url.Error
		if errors.As(err, &urlError) {
			err = urlError.Err // Strip query strings, which carry Firebase's API key.
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		defer response.Body.Close()
		return nil, responseError(response)
	}
	return response, nil
}

func responseError(response *http.Response) error {
	var failure struct {
		Detail struct {
			Status string `json:"status"`
		} `json:"detail"`
	}
	_ = json.NewDecoder(io.LimitReader(response.Body, 65536)).Decode(&failure)
	if failure.Detail.Status == "detected_unusual_activity" {
		return &Error{Code: 4, Status: response.StatusCode, Message: "ElevenLabs disabled free-tier generation for this connection; generation cannot continue while this restriction applies"}
	}
	code := 6
	switch response.StatusCode {
	case 401:
		code = 3
	case 403:
		code = 4
	case 429:
		code = 5
	}
	// Server responses can contain account credentials or reflected request text.
	// Report only known status metadata; never dump a response body to stderr.
	message := fmt.Sprintf("HTTP %d from %s", response.StatusCode, response.Request.URL.EscapedPath())
	switch code {
	case 3:
		message += "; import a fresh browser session with auth login"
	case 4:
		message += "; this account cannot perform the operation"
	case 5:
		message += "; rate limited, do not retry immediately"
		if after := response.Header.Get("Retry-After"); after != "" {
			message += " (Retry-After: " + after + ")"
		}
	}
	return &Error{Code: code, Status: response.StatusCode, Message: message}
}

// Refresh follows the Firebase SDK's form-encoded token exchange. Persist before
// making an API call so a later API failure does not discard rotated credentials.
func (c *Client) Refresh(ctx context.Context) error {
	if c.Session == nil {
		return &Error{Code: 3, Message: "no session; run elevenlabs auth login --session FILE"}
	}
	endpoint, err := url.Parse(c.TokenURL)
	if err != nil {
		return err
	}
	query := endpoint.Query()
	query.Set("key", c.Session.APIKey)
	endpoint.RawQuery = query.Encode()
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {c.Session.RefreshToken}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := c.send(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    string `json:"expires_in"`
	}
	if err := json.NewDecoder(response.Body).Decode(&tokens); err != nil {
		return fmt.Errorf("invalid token response: %w", err)
	}
	seconds, err := strconv.Atoi(tokens.ExpiresIn)
	if err != nil || seconds <= 0 || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		return &Error{Code: 6, Message: "token response missing valid tokens or expiry"}
	}
	updated := Session{UserAgent: c.Session.UserAgent, APIKey: c.Session.APIKey, AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, ExpiresAt: time.Now().Add(time.Duration(seconds) * time.Second)}
	if c.Save != nil {
		if err := c.Save(&updated); err != nil {
			return fmt.Errorf("save renewed session: %w", err)
		}
	}
	*c.Session = updated
	return nil
}

func (c *Client) Request(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	if c.Session == nil {
		return nil, &Error{Code: 3, Message: "no session; run elevenlabs auth login --session FILE"}
	}
	if time.Until(c.Session.ExpiresAt) < 30*time.Second {
		if err := c.Refresh(ctx); err != nil {
			return nil, err
		}
	}
	relative, err := url.Parse(path)
	if err != nil || relative.IsAbs() || relative.Host != "" || !(strings.HasPrefix(path, "/v1/") || path == "/sound-generation") {
		return nil, &Error{Code: 2, Message: "expected a supported ElevenLabs API path"}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Session.AccessToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.send(req)
}

func (c *Client) JSON(ctx context.Context, method, path string, input any) (any, error) {
	var body io.Reader
	if input != nil {
		raw, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(raw)
	}
	response, err := c.Request(ctx, method, path, body, "application/json")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNoContent {
		return map[string]bool{"ok": true}, nil
	}
	var value any
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, &Error{Code: 6, Message: "API returned invalid JSON"}
	}
	return value, nil
}
