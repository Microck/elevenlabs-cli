package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Microck/elevenlabs-web-cli/internal/elevenlabs"
	"github.com/Microck/elevenlabs-web-cli/internal/filelock"
	"github.com/spf13/cobra"
)

type localConfig struct {
	DefaultProfile string `json:"default_profile"`
}

func (a *app) configPath() string { return filepath.Join(a.dir, "config.json") }

func (a *app) loadConfig() (localConfig, error) {
	config := localConfig{DefaultProfile: "default"}
	raw, err := os.ReadFile(a.configPath())
	if os.IsNotExist(err) {
		return config, nil
	}
	if err != nil {
		return config, err
	}
	if err = json.Unmarshal(raw, &config); err != nil {
		return config, fmt.Errorf("invalid config.json: %w", err)
	}
	if _, err = elevenlabs.SessionPath(a.dir, config.DefaultProfile); err != nil {
		return config, fmt.Errorf("invalid default_profile: %w", err)
	}
	return config, nil
}

func (a *app) selectProfile(name string) error {
	path, err := elevenlabs.SessionPath(a.dir, name)
	if err != nil {
		return err
	}
	if _, err = elevenlabs.LoadSession(path); err != nil {
		return err
	}
	// Read before writing so a malformed config is never silently discarded.
	config, err := a.loadConfig()
	if err != nil {
		return err
	}
	config.DefaultProfile = name
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err = elevenlabs.WritePrivateFile(a.configPath(), raw); err != nil {
		return err
	}
	return a.print(config)
}

func (a *app) configCommand() *cobra.Command {
	parent := &cobra.Command{Use: "config", Short: "Inspect and change local CLI settings"}
	parent.AddCommand(
		&cobra.Command{Use: "path", Short: "Show the local configuration path", Args: exact(0), RunE: func(_ *cobra.Command, _ []string) error { return a.print(map[string]string{"path": a.configPath()}) }},
		&cobra.Command{Use: "list", Short: "Show local settings without credentials", Args: exact(0), RunE: func(_ *cobra.Command, _ []string) error {
			config, err := a.loadConfig()
			if err != nil {
				return err
			}
			return a.print(config)
		}},
		&cobra.Command{Use: "get KEY", Short: "Read default_profile", Args: exact(1), RunE: func(_ *cobra.Command, args []string) error {
			if args[0] != "default_profile" {
				return usage("unknown config key; supported key: default_profile")
			}
			config, err := a.loadConfig()
			if err != nil {
				return err
			}
			return a.print(config.DefaultProfile)
		}},
		&cobra.Command{Use: "set KEY VALUE", Short: "Set default_profile to an imported profile", Args: exact(2), RunE: func(_ *cobra.Command, args []string) error {
			if args[0] != "default_profile" {
				return usage("unknown config key; supported key: default_profile")
			}
			return a.selectProfile(args[1])
		}},
	)
	return parent
}

func (a *app) removeProfile(ctx context.Context, name string, yes bool) error {
	if !yes {
		return usage("pass --yes to remove this local session")
	}
	path, err := elevenlabs.SessionPath(a.dir, name)
	if err != nil {
		return err
	}
	lock, err := a.lockProfile(ctx, path)
	if err != nil {
		return err
	}
	defer filelock.Release(lock)
	session, err := elevenlabs.LoadSession(path)
	if err != nil {
		var auth *elevenlabs.Error
		if !errors.As(err, &auth) || auth.Code != 3 {
			return err
		}
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			return err
		}
	} else {
		engine, err := a.openEngine(ctx, session.APIKey)
		if err != nil {
			return err
		}
		defer engine.Close()
		if err = cleanupRetired(ctx, path, session, engine); err != nil {
			return err
		}
		if err = engine.Forget(ctx, session.GalletonID); err != nil {
			return elevenlabs.ManagedError(err)
		}
	}
	if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return a.print(map[string]any{"profile": name, "removed": true})
}
