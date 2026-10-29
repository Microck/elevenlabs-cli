package elevenlabs

import (
	"encoding/json"
	"net/url"
)

// Adapter describes the observed Firebase form exchange. The application key is
// public browser configuration; refresh tokens and user agents stay in the vault.
func Adapter(apiKey string) []byte {
	config := map[string]any{"providers": []any{map[string]any{
		"name": "elevenlabs", "kind": "http",
		"origins":        []string{"https://securetoken.googleapis.com", APIBase},
		"refresh_url":    TokenURL + "?key=" + url.QueryEscape(apiKey),
		"refresh_method": "POST", "body_format": "form",
		"body":                     map[string]string{"grant_type": "refresh_token", "refresh_token": "${refresh_token}"},
		"refresh_headers":          map[string]string{"Origin": "https://elevenlabs.io", "Referer": "https://elevenlabs.io/", "User-Agent": "${secret.user_agent}"},
		"response":                 map[string]string{"access_token_pointer": "/access_token", "refresh_token_pointer": "/refresh_token", "expires_in_pointer": "/expires_in"},
		"refresh_interval_seconds": 3600, "refresh_before_seconds": 60, "timeout_seconds": 120,
	}}}
	raw, _ := json.Marshal(config)
	return raw
}
