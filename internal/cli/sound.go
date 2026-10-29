package cli

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/Microck/elevenlabs-cli/internal/elevenlabs"
	"github.com/spf13/cobra"
)

func (a *app) sound() *cobra.Command {
	parent := &cobra.Command{Use: "sound", Short: "Generate and download sound effects"}
	var input, model, output string
	var duration, influence float64
	var variants int
	var loop bool
	generate := &cobra.Command{Use: "generate", Short: "Generate sound effects (consumes credits)", Args: exact(0), RunE: func(cmd *cobra.Command, _ []string) error {
		if math.IsNaN(duration) || math.IsInf(duration, 0) || duration <= 0 {
			return usage("--duration must be a positive number of seconds")
		}
		if err := unitInterval("prompt-influence", influence); err != nil {
			return err
		}
		if variants < 1 || variants > 4 {
			return usage("--variants must be between 1 and 4")
		}
		if strings.TrimSpace(model) == "" {
			return usage("--model cannot be empty")
		}
		raw, err := readInput(input, a.stdin)
		if err != nil {
			return err
		}
		if !utf8.Valid(raw) || strings.TrimSpace(string(raw)) == "" {
			return usage("sound prompt must be nonempty UTF-8 text")
		}
		if err := a.connect(); err != nil {
			return err
		}
		// This web workflow can publish to Explore through an account setting.
		// Require a verified opt-out rather than changing the account silently.
		config, err := a.client.JSON(cmd.Context(), "GET", "/v1/sfx-user-config", nil)
		if err != nil {
			return err
		}
		object, ok := config.(map[string]any)
		if !ok {
			return &elevenlabs.Error{Code: 6, Message: "unexpected sound sharing configuration"}
		}
		sharing, ok := object["auto_explore_sharing_opt_in"].(bool)
		if !ok || sharing {
			return &elevenlabs.Error{Code: 4, Message: "disable Explore sharing in the ElevenLabs Sound Effects page before generating"}
		}
		payload := map[string]any{"text": string(raw), "prompt_influence": influence, "duration_seconds": duration, "loop": loop, "number_of_generations": variants, "model_id": model}
		value, err := a.client.JSON(cmd.Context(), "POST", "/sound-generation", payload)
		if err != nil {
			return err
		}
		return a.print(value)
	}}
	generate.Flags().StringVar(&input, "text-file", "", "UTF-8 prompt file, or - for stdin")
	generate.Flags().StringVar(&model, "model", "eleven_text_to_sound_v3", "model ID")
	generate.Flags().Float64Var(&duration, "duration", 1, "duration in seconds")
	generate.Flags().Float64Var(&influence, "prompt-influence", 0.3, "prompt influence (0-1)")
	generate.Flags().IntVar(&variants, "variants", 4, "number of variants (1-4)")
	generate.Flags().BoolVar(&loop, "loop", false, "generate a loop")
	download := &cobra.Command{Use: "download ID", Short: "Download a generated sound as Opus", Args: exact(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkID(args[0]); err != nil {
			return err
		}
		return a.audio(cmd, "GET", "/v1/sound-generation/history/"+args[0]+"/audio", nil, output)
	}}
	download.Flags().StringVar(&output, "output", "", "new audio file, or - for binary stdout")
	parent.AddCommand(generate, download)
	return parent
}
