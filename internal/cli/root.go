package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Microck/elevenlabs-web-cli/internal/elevenlabs"
	"github.com/spf13/cobra"
)

type app struct {
	stdin                             io.Reader
	stdout, stderr                    io.Writer
	dir, profile, format, errorFormat string
	noInput, debug                    bool
	client                            *elevenlabs.Client
}

func Execute(ctx context.Context, version string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	a := &app{stdin: stdin, stdout: stdout, stderr: stderr}
	root := a.root(version)
	root.SetArgs(args)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	cmd, err := root.ExecuteContextC(ctx)
	if err == nil {
		return 0
	}
	// Cobra stops before parsing flags when command lookup fails. Parse them
	// here so usage errors still honor --error-format without running a command.
	if cmd != nil && cmd.CalledAs() == "" {
		if parseErr := cmd.ParseFlags(args); parseErr != nil {
			err = parseErr
		}
		err = usage(err.Error())
	}
	failure := &elevenlabs.Error{Code: 1, Message: err.Error()}
	var apiError *elevenlabs.Error
	if errors.As(err, &apiError) {
		failure = apiError
	}
	if errors.Is(err, context.Canceled) || ctx.Err() != nil {
		failure = &elevenlabs.Error{Code: 130, Message: "interrupted"}
	}
	if a.errorFormat == "json" {
		_ = json.NewEncoder(stderr).Encode(failure)
	} else {
		fmt.Fprintln(stderr, "elevenlabs: "+failure.Message)
	}
	return failure.Code
}

func usage(message string) error { return &elevenlabs.Error{Code: 2, Message: message} }
func exact(count int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(count)(cmd, args); err != nil {
			return usage(err.Error())
		}
		return nil
	}
}

func (a *app) root(version string) *cobra.Command {
	root := &cobra.Command{Use: "elevenlabs", Short: "Use ElevenLabs from the terminal with your web account", Version: version, SilenceUsage: true, SilenceErrors: true}
	root.Long = "Use ElevenLabs with an imported browser session. Output is JSON by default.\nUse --format pretty for indented output. Errors go to stderr."
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usage(err.Error()) })
	root.PersistentFlags().StringVar(&a.profile, "profile", "default", "account profile")
	root.PersistentFlags().StringVar(&a.format, "format", "json", "json, jsonl or pretty")
	root.PersistentFlags().StringVar(&a.errorFormat, "error-format", "text", "text or json")
	root.PersistentFlags().BoolVar(&a.noInput, "no-input", false, "never prompt")
	root.PersistentFlags().BoolVar(&a.debug, "debug", false, "log request metadata to stderr")
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		switch a.format {
		case "json", "jsonl", "pretty":
		default:
			return usage("--format must be json, jsonl or pretty")
		}
		if a.errorFormat != "text" && a.errorFormat != "json" {
			return usage("--error-format must be text or json")
		}
		a.dir = os.Getenv("ELEVENLABS_CONFIG_DIR")
		if a.dir == "" {
			dir, err := os.UserConfigDir()
			if err != nil {
				return err
			}
			a.dir = filepath.Join(dir, "elevenlabs-web-cli")
		}
		if !cmd.Flags().Changed("profile") && cmd.CommandPath() != "elevenlabs config path" {
			config, err := a.loadConfig()
			if err != nil {
				return err
			}
			a.profile = config.DefaultProfile
		}
		_, err := elevenlabs.SessionPath(a.dir, a.profile)
		return err
	}
	root.AddCommand(a.auth(), a.profiles(), a.me(), a.models(), a.voices(), a.history(), a.speech(false), a.speech(true), a.sound(), a.configCommand(), a.skills(), a.doctor())
	// Cobra otherwise shows help with exit 0 for unknown subcommands of groups.
	for _, group := range root.Commands() {
		if group.HasSubCommands() {
			group.Args = exact(0)
			group.RunE = func(cmd *cobra.Command, _ []string) error { return cmd.Help() }
		}
	}
	return root
}

func (a *app) connect() error {
	if a.client != nil {
		return nil
	}
	path, err := elevenlabs.SessionPath(a.dir, a.profile)
	if err != nil {
		return err
	}
	session, err := elevenlabs.LoadSession(path)
	if err != nil {
		return err
	}
	a.client = elevenlabs.New(session)
	a.client.Save = func(s *elevenlabs.Session) error { return elevenlabs.SaveSession(path, s) }
	if a.debug {
		a.client.Debug = a.stderr
	}
	return nil
}

func (a *app) print(value any) error {
	encoder := json.NewEncoder(a.stdout)
	if a.format == "pretty" {
		encoder.SetIndent("", "  ")
	}
	if a.format == "jsonl" {
		if items, ok := value.([]any); ok {
			for _, item := range items {
				if err := encoder.Encode(item); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return encoder.Encode(value)
}

func (a *app) readCommand(use, short, path, collection string) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Args: exact(0), RunE: func(cmd *cobra.Command, _ []string) error {
		if err := a.connect(); err != nil {
			return err
		}
		value, err := a.client.JSON(cmd.Context(), "GET", path, nil)
		if err != nil {
			return err
		}
		if collection != "" {
			object, ok := value.(map[string]any)
			if !ok {
				return &elevenlabs.Error{Code: 6, Message: "API collection shape changed"}
			}
			value, ok = object[collection]
			if !ok {
				return &elevenlabs.Error{Code: 6, Message: "API response missing " + collection}
			}
		}
		return a.print(value)
	}}
}

func (a *app) me() *cobra.Command {
	cmd := &cobra.Command{Use: "me", Short: "Inspect your account and allowance"}
	cmd.AddCommand(a.readCommand("show", "Account details", "/v1/user/internal", ""), a.readCommand("subscription", "Plan and credit allowance", "/v1/user/subscription", ""))
	return cmd
}
func (a *app) models() *cobra.Command {
	cmd := &cobra.Command{Use: "model", Short: "Discover available models"}
	cmd.AddCommand(a.readCommand("list", "List models and capabilities", "/v1/models", ""))
	return cmd
}
func (a *app) voices() *cobra.Command {
	cmd := &cobra.Command{Use: "voice", Short: "Browse voices"}
	cmd.AddCommand(
		a.readCommand("list", "List your available voices", "/v1/voices", "voices"),
		a.readCommand("collections", "List your voice collections", "/v1/voices/collections", "collections"),
		a.voiceSearch(),
		a.resourceRead("show ID", "Show voice details", "/v1/voices/", ""),
		a.resourceRead("settings ID", "Show voice settings", "/v1/voices/", "/settings"),
		a.readCommand("defaults", "Default voice settings", "/v1/voices/settings/default", ""),
	)
	return cmd
}
func (a *app) profiles() *cobra.Command {
	cmd := &cobra.Command{Use: "profile", Short: "List local accounts"}
	cmd.AddCommand(&cobra.Command{Use: "list", Args: exact(0), RunE: func(_ *cobra.Command, _ []string) error {
		names, err := elevenlabs.Profiles(a.dir)
		if err != nil {
			return err
		}
		items := make([]any, len(names))
		for i, name := range names {
			items[i] = name
		}
		return a.print(items)
	}})
	var yes bool
	remove := &cobra.Command{Use: "remove NAME", Short: "Remove one local profile session", Args: exact(1), RunE: func(_ *cobra.Command, args []string) error { return a.removeProfile(args[0], yes) }}
	remove.Flags().BoolVar(&yes, "yes", false, "confirm removal")
	cmd.AddCommand(remove, &cobra.Command{Use: "use NAME", Short: "Select an imported profile for future commands", Args: exact(1), RunE: func(_ *cobra.Command, args []string) error { return a.selectProfile(args[0]) }})
	return cmd
}
func (a *app) doctor() *cobra.Command {
	return &cobra.Command{Use: "doctor", Short: "Check session renewal and account access", Args: exact(0), RunE: func(cmd *cobra.Command, _ []string) error {
		if err := a.connect(); err != nil {
			return err
		}
		if err := a.client.Refresh(cmd.Context()); err != nil {
			return err
		}
		if _, err := a.client.JSON(cmd.Context(), "GET", "/v1/user/subscription", nil); err != nil {
			return err
		}
		return a.print(map[string]any{"ok": true, "profile": a.profile, "checks": []string{"session_refresh", "account_access"}})
	}}
}

func readInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(io.LimitReader(stdin, 2<<20))
	}
	if strings.TrimSpace(path) == "" {
		return nil, usage("provide a file or - for stdin")
	}
	return os.ReadFile(path)
}
