package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Microck/elevenlabs-web-cli/internal/elevenlabs"
	"github.com/Microck/elevenlabs-web-cli/internal/testengine"
)

func TestSpeechDownloadWorkflow(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/v1/text-to-dialogue/stream" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		expected := `{"inputs":[{"text":"Hi.","voice_id":"Xb7hH8MSUJpSbSDYk0k2"}],"model_id":"eleven_v4_exp","settings":{"similarity":0.75,"stability":0.5,"text_focus":false}}`
		actual, _ := json.Marshal(body)
		if string(actual) != expected {
			t.Errorf("payload %s", actual)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		fmt.Fprint(w, "ID3fixture")
	}))
	defer server.Close()
	var output bytes.Buffer
	client := elevenlabs.New(&elevenlabs.Session{GalletonID: "fixture", UserAgent: "Mozilla/5.0 fixture"})
	client.BaseURL = server.URL
	client.Engine = testengine.Fixture(t, elevenlabs.Adapter("fixture-key"), server.URL).Client
	a := &app{stdin: strings.NewReader("Hi."), stdout: &output, stderr: &bytes.Buffer{}, client: client}
	cmd := a.speech(false)
	dest := filepath.Join(t.TempDir(), "speech.mp3")
	cmd.SetArgs([]string{"generate", "--text-file", "-", "--voice", "Xb7hH8MSUJpSbSDYk0k2", "--output", dest})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	audio, err := os.ReadFile(dest)
	if err != nil || string(audio) != "ID3fixture" {
		t.Fatalf("audio %q %v", audio, err)
	}
	if !json.Valid(output.Bytes()) {
		t.Fatalf("stdout %q", output.String())
	}
	a.stdin = strings.NewReader("Hi.")
	if err := cmd.Execute(); err == nil {
		t.Fatal("overwrote existing output")
	}
	if calls != 1 {
		t.Fatalf("existing output triggered request: %d", calls)
	}
}

func TestAudioFailureRemovesPartialFile(t *testing.T) {
	for _, mode := range []string{"restriction", "wrong-type", "empty", "truncated"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch mode {
				case "restriction":
					w.WriteHeader(401)
					fmt.Fprint(w, `{"detail":{"status":"detected_unusual_activity"}}`)
				case "wrong-type":
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"ok":true}`)
				case "empty":
					w.Header().Set("Content-Type", "audio/mpeg")
				case "truncated":
					w.Header().Set("Content-Type", "audio/mpeg")
					w.Header().Set("Content-Length", "99")
					fmt.Fprint(w, "partial")
				}
			}))
			defer server.Close()
			client := elevenlabs.New(&elevenlabs.Session{GalletonID: "fixture", UserAgent: "Mozilla/5.0 fixture"})
			client.BaseURL = server.URL
			client.Engine = testengine.Fixture(t, elevenlabs.Adapter("fixture-key"), server.URL).Client
			var stdout bytes.Buffer
			a := &app{stdout: &stdout, client: client}
			dest := filepath.Join(t.TempDir(), "audio.mp3")
			cmd := a.historyDownload()
			cmd.SetArgs([]string{"fixture-id", "--output", dest})
			if err := cmd.Execute(); err == nil {
				t.Fatal("accepted failed audio")
			}
			if _, err := os.Stat(dest); !os.IsNotExist(err) {
				t.Fatalf("partial output survived: %v", err)
			}
			if stdout.Len() != 0 {
				t.Fatal("success output on failure")
			}
		})
	}
}

func TestDialogueValidationBeforeNetwork(t *testing.T) {
	for _, input := range []string{`[]`, `null`, `[{"text":"","voice_id":"id"}]`, `[{"text":"Hi","voice_id":"../bad"}]`, `[{"text":"Hi","voice_id":"id","unknown":1}]`, `[] {}`} {
		a := &app{stdin: strings.NewReader(input), stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
		cmd := a.speech(true)
		cmd.SetArgs([]string{"generate", "--input", "-", "--output", "-"})
		if err := cmd.Execute(); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
}

func TestHistoryBinaryStdout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		raw, _ := json.Marshal(payload)
		if r.URL.Path != "/v1/history/download" || string(raw) != `{"history_item_ids":["sample-id"],"output_format":"mp3"}` {
			t.Errorf("request %s %s", r.URL.Path, raw)
		}
		w.Header().Set("Content-Type", "audio/mpeg3")
		fmt.Fprint(w, "ID3fixture")
	}))
	defer server.Close()
	client := elevenlabs.New(&elevenlabs.Session{GalletonID: "fixture", UserAgent: "Mozilla/5.0 fixture"})
	client.BaseURL = server.URL
	client.Engine = testengine.Fixture(t, elevenlabs.Adapter("fixture-key"), server.URL).Client
	var stdout bytes.Buffer
	a := &app{stdout: &stdout, client: client}
	cmd := a.historyDownload()
	cmd.SetArgs([]string{"sample-id", "--output", "-"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "ID3fixture" {
		t.Fatalf("binary stdout polluted: %q", stdout.String())
	}
}

func TestSoundSharingGateAndCapturedPayload(t *testing.T) {
	for _, sharing := range []bool{true, false} {
		t.Run(fmt.Sprint(sharing), func(t *testing.T) {
			generations := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v1/sfx-user-config" {
					fmt.Fprintf(w, `{"auto_explore_sharing_opt_in":%t}`, sharing)
					return
				}
				generations++
				if r.Method != "POST" || r.URL.Path != "/sound-generation" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				raw, _ := json.Marshal(payload)
				if string(raw) != `{"duration_seconds":1,"loop":false,"model_id":"eleven_text_to_sound_v3","number_of_generations":4,"prompt_influence":0.3,"text":"One short wooden tap."}` {
					t.Errorf("payload %s", raw)
				}
				fmt.Fprint(w, `{"sound_generations_with_waveforms":[]}`)
			}))
			defer server.Close()
			client := elevenlabs.New(&elevenlabs.Session{GalletonID: "fixture", UserAgent: "Mozilla/5.0 fixture"})
			client.BaseURL = server.URL
			client.Engine = testengine.Fixture(t, elevenlabs.Adapter("fixture-key"), server.URL).Client
			a := &app{stdin: strings.NewReader("One short wooden tap."), stdout: &bytes.Buffer{}, client: client}
			cmd := a.sound()
			cmd.SetArgs([]string{"generate", "--text-file", "-"})
			err := cmd.Execute()
			if sharing && (err == nil || generations != 0) {
				t.Fatalf("sharing gate: %v, generations %d", err, generations)
			}
			if !sharing && (err != nil || generations != 1) {
				t.Fatalf("generation: %v, generations %d", err, generations)
			}
		})
	}
}
