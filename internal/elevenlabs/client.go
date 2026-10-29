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
	"strings"
	"time"

	"github.com/Microck/elevenlabs-web-cli/internal/galleton"
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
	HTTP       *http.Client
	BaseURL    string
	Session    *Session
	Debug      io.Writer
	Engine     *galleton.Client
	OpenEngine func(context.Context) (*galleton.Client, error)
}

func New(session *Session) *Client {
	return &Client{HTTP: &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}, BaseURL: APIBase, Session: session}
}
func (c *Client) engine(ctx context.Context) (*galleton.Client, error) {
	if c.Engine == nil {
		if c.OpenEngine == nil {
			return nil, &Error{Code: 3, Message: "no managed session; run auth login"}
		}
		engine, err := c.OpenEngine(ctx)
		if err != nil {
			return nil, err
		}
		c.Engine = engine
	}
	return c.Engine, nil
}
func (c *Client) Close() {
	if c.Engine != nil {
		c.Engine.Close()
	}
}
func ManagedError(err error) error {
	if err == nil {
		return nil
	}
	var failure *galleton.Error
	if !errors.As(err, &failure) {
		return err
	}
	code := 1
	message := "Galleton could not complete the operation; inspect auth service status before retrying"
	switch {
	case failure.Code == "reauth_required" || failure.Code == "renewal_uncertain" || failure.Status == 404:
		code = 3
		message = "Galleton needs a fresh browser session; run auth login"
	case failure.Status == 401 || failure.Status == 403:
		code = 3
		message = "Galleton rejected local access or the provider origin"
	case failure.Status == 429:
		code = 5
		message = "Galleton is rate limited; wait before retrying"
	}
	return &Error{Code: code, Message: message}
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

// Refresh delegates rotation and persistence to the shared engine.
func (c *Client) Refresh(ctx context.Context) error {
	engine, err := c.engine(ctx)
	if err != nil {
		return err
	}
	_, err = engine.Refresh(ctx, c.Session.GalletonID)
	return ManagedError(err)
}

func (c *Client) Request(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	if c.Session == nil {
		return nil, &Error{Code: 3, Message: "no session; run elevenlabs auth login --session FILE"}
	}
	relative, err := url.Parse(path)
	if err != nil || relative.IsAbs() || relative.Host != "" || !(strings.HasPrefix(path, "/v1/") || path == "/sound-generation") {
		return nil, &Error{Code: 2, Message: "expected a supported ElevenLabs API path"}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	engine, err := c.engine(ctx)
	if err != nil {
		return nil, err
	}
	headers, err := engine.Headers(ctx, c.Session.GalletonID, c.BaseURL)
	if err != nil {
		return nil, ManagedError(err)
	}
	authorization := headers.Headers["Authorization"]
	if !strings.HasPrefix(authorization, "Bearer ") || len(authorization) <= 7 || strings.ContainsAny(authorization, "\r\n") {
		return nil, &Error{Code: 6, Message: "Galleton returned no valid bearer header"}
	}
	req.Header.Set("Authorization", authorization)
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
