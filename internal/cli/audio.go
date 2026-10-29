package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Microck/elevenlabs-web-cli/internal/elevenlabs"
	"github.com/spf13/cobra"
)

var resourceID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func checkID(id string) error {
	if !resourceID.MatchString(id) {
		return usage("ID must contain only letters, numbers, underscores or hyphens")
	}
	return nil
}

type dialogueInput struct {
	Text    string `json:"text"`
	VoiceID string `json:"voice_id"`
}

func (a *app) speech(dialogue bool) *cobra.Command {
	name := "speech"
	if dialogue {
		name = "dialogue"
	}
	parent := &cobra.Command{Use: name, Short: "Generate speech audio using your web account"}
	var input, voice, output, model string
	var stability, similarity float64
	var textFocus bool
	cmd := &cobra.Command{Use: "generate", Short: "Generate MP3 audio (consumes credits)", Args: exact(0), RunE: func(cmd *cobra.Command, _ []string) error {
		if err := unitInterval("stability", stability); err != nil {
			return err
		}
		if err := unitInterval("similarity", similarity); err != nil {
			return err
		}
		if strings.TrimSpace(model) == "" {
			return usage("--model cannot be empty")
		}
		raw, err := readInput(input, a.stdin)
		if err != nil {
			return err
		}
		if !utf8.Valid(raw) {
			return usage("input must be UTF-8")
		}
		var inputs []dialogueInput
		if dialogue {
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&inputs); err != nil {
				return usage("--input must contain an array of {text, voice_id} objects")
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				return usage("--input must contain exactly one JSON array")
			}
		} else {
			inputs = []dialogueInput{{Text: string(raw), VoiceID: voice}}
		}
		if len(inputs) == 0 {
			return usage("provide at least one dialogue input")
		}
		for _, item := range inputs {
			if strings.TrimSpace(item.Text) == "" {
				return usage("speech text cannot be empty")
			}
			if err := checkID(item.VoiceID); err != nil {
				return err
			}
		}
		payload := map[string]any{"inputs": inputs, "model_id": model, "settings": map[string]any{"stability": stability, "similarity": similarity, "text_focus": textFocus}}
		return a.audio(cmd, "POST", "/v1/text-to-dialogue/stream", payload, output)
	}}
	if dialogue {
		cmd.Flags().StringVar(&input, "input", "", "dialogue JSON file, or - for stdin")
	} else {
		cmd.Flags().StringVar(&input, "text-file", "", "UTF-8 text file, or - for stdin")
		cmd.Flags().StringVar(&voice, "voice", "", "stock or account voice ID")
	}
	cmd.Flags().StringVar(&output, "output", "", "new audio file, or - for binary stdout")
	cmd.Flags().StringVar(&model, "model", "eleven_v4_exp", "model ID")
	cmd.Flags().Float64Var(&stability, "stability", 0.5, "stability (0-1)")
	cmd.Flags().Float64Var(&similarity, "similarity", 0.75, "similarity (0-1)")
	cmd.Flags().BoolVar(&textFocus, "text-focus", false, "enable text focus")
	parent.AddCommand(cmd)
	return parent
}

func unitInterval(name string, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
		return usage("--" + name + " must be between 0 and 1")
	}
	return nil
}

// Reserve the destination before sending requests that may consume credits.
// Failed writes remove only this command's newly created file; stdout cannot roll back.
func (a *app) audio(cmd *cobra.Command, method, path string, input any, output string) (err error) {
	if strings.TrimSpace(output) == "" {
		return usage("--output is required; use - for binary stdout")
	}
	writer := a.stdout
	var file *os.File
	complete := false
	if output != "-" {
		file, err = os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return fmt.Errorf("create audio output: %w", err)
		}
		defer func() {
			file.Close()
			if !complete {
				os.Remove(output)
			}
		}()
		writer = file
	}
	if err = a.connect(); err != nil {
		return err
	}
	var body io.Reader
	if input != nil {
		raw, marshalErr := json.Marshal(input)
		if marshalErr != nil {
			return marshalErr
		}
		body = bytes.NewReader(raw)
	}
	response, err := a.client.Request(cmd.Context(), method, path, body, "application/json")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	mediaType, _, parseErr := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if parseErr != nil || (!strings.HasPrefix(mediaType, "audio/") && mediaType != "application/octet-stream") {
		return &elevenlabs.Error{Code: 6, Message: "expected an audio response"}
	}
	count, err := io.Copy(writer, response.Body)
	if err != nil {
		return fmt.Errorf("audio download interrupted: %w", err)
	}
	if count == 0 {
		return &elevenlabs.Error{Code: 6, Message: "server returned empty audio"}
	}
	if file != nil {
		if err = file.Close(); err != nil {
			return err
		}
		// A reporting failure must not delete a completely downloaded audio file.
		complete = true
		return a.print(map[string]any{"output": output, "bytes": count})
	}
	return nil
}

func (a *app) historyDownload() *cobra.Command {
	var output string
	cmd := &cobra.Command{Use: "download ID", Short: "Download one history item as MP3", Args: exact(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkID(args[0]); err != nil {
			return err
		}
		return a.audio(cmd, "POST", "/v1/history/download", map[string]any{"history_item_ids": []string{args[0]}, "output_format": "mp3"}, output)
	}}
	cmd.Flags().StringVar(&output, "output", "", "new audio file, or - for binary stdout")
	return cmd
}
