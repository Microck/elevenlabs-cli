package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Microck/elevenlabs-web-cli/internal/elevenlabs"
	"github.com/Microck/elevenlabs-web-cli/internal/filelock"
	"github.com/Microck/elevenlabs-web-cli/internal/galleton"
	"github.com/Microck/elevenlabs-web-cli/scripts"
	"github.com/spf13/cobra"
)

func (a *app) openEngine(ctx context.Context, apiKey string) (*galleton.Client, error) {
	dir, err := filepath.Abs(filepath.Join(a.dir, "galleton"))
	if err != nil {
		return nil, err
	}
	return galleton.Open(ctx, dir, galleton.Setup{Adapter: elevenlabs.Adapter(apiKey), Launched: func(c *galleton.Client) error { return c.Resume(ctx, "elevenlabs") }})
}

func (a *app) auth() *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Import and manage a Galleton browser session"}
	var input string
	login := &cobra.Command{Use: "login", Short: "Import a browser export into the managed session engine", Args: exact(0), RunE: func(cmd *cobra.Command, _ []string) error {
		raw, err := readInput(input, a.stdin)
		if err != nil {
			return err
		}
		var browser elevenlabs.BrowserSession
		if json.Unmarshal(raw, &browser) != nil {
			return usage("invalid session JSON; use auth export-script in the browser")
		}
		if err = browser.Validate(); err != nil {
			return err
		}
		path, err := elevenlabs.SessionPath(a.dir, a.profile)
		if err != nil {
			return err
		}
		lock, err := a.lockProfile(cmd.Context(), path)
		if err != nil {
			return err
		}
		defer filelock.Release(lock)
		// Read the prior reference before changing anything. Old plaintext exports
		// are replaced only by this explicit login, never silently imported.
		var prior elevenlabs.Session
		if old, readErr := os.ReadFile(path); readErr == nil {
			if json.Unmarshal(old, &prior) != nil {
				return fmt.Errorf("invalid profile JSON")
			}
		} else if !os.IsNotExist(readErr) {
			return readErr
		}
		engine, err := a.openEngine(cmd.Context(), browser.APIKey)
		if err != nil {
			return err
		}
		defer engine.Close()
		var random [16]byte
		if _, err = rand.Read(random[:]); err != nil {
			return err
		}
		id := "elevenlabs-" + hex.EncodeToString(random[:])
		saved := false
		defer func() {
			if !saved {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = engine.Forget(ctx, id)
			}
		}()
		_, err = engine.Connect(cmd.Context(), id, galleton.Credentials{Provider: "elevenlabs", AccessToken: browser.AccessToken, RefreshToken: browser.RefreshToken, AccessExpiresAt: &browser.ExpiresAt, Secrets: map[string]string{"user_agent": browser.UserAgent}})
		if err != nil {
			return elevenlabs.ManagedError(err)
		}
		session := &elevenlabs.Session{GalletonID: id, APIKey: browser.APIKey, UserAgent: browser.UserAgent, RetiredIDs: prior.RetiredIDs}
		if prior.GalletonID != "" {
			session.RetiredIDs = append(session.RetiredIDs, prior.GalletonID)
		}
		client := a.newClient(session)
		client.Engine = engine
		if err = client.Refresh(cmd.Context()); err != nil {
			return err
		}
		if _, err = client.JSON(cmd.Context(), "GET", "/v1/user/internal", nil); err != nil {
			return err
		}
		if err = elevenlabs.SaveSession(path, session); err != nil {
			return err
		}
		saved = true
		if err = cleanupRetired(cmd.Context(), path, session, engine); err != nil {
			return fmt.Errorf("new login saved; retired session cleanup will retry on next authenticated command: %w", err)
		}
		return a.print(map[string]any{"profile": a.profile, "imported": true, "source": "galleton", "galleton_id": id})
	}}
	login.Flags().StringVar(&input, "session", "", "browser export JSON file, or - for stdin")
	var check bool
	status := &cobra.Command{Use: "status", Short: "Show the saved session reference without credentials", Args: exact(0), RunE: func(cmd *cobra.Command, _ []string) error {
		if err := a.connect(); err != nil {
			return err
		}
		result := map[string]any{"profile": a.profile, "source": "galleton", "galleton_id": a.client.Session.GalletonID}
		if check {
			if _, err := a.client.JSON(cmd.Context(), "GET", "/v1/user/internal", nil); err != nil {
				return err
			}
			meta, err := a.client.Engine.Status(cmd.Context(), a.client.Session.GalletonID)
			if err != nil {
				return elevenlabs.ManagedError(err)
			}
			result["status"], result["expires_at"] = meta.Status, meta.AccessExpiresAt
		}
		return a.print(result)
	}}
	status.Flags().BoolVar(&check, "check", false, "check the engine and account access")
	refresh := &cobra.Command{Use: "refresh", Short: "Renew the session through Galleton", Args: exact(0), RunE: func(cmd *cobra.Command, _ []string) error {
		if err := a.connect(); err != nil {
			return err
		}
		if err := a.client.Refresh(cmd.Context()); err != nil {
			return err
		}
		return a.print(map[string]any{"profile": a.profile, "refreshed": true, "source": "galleton"})
	}}
	var yes bool
	logout := &cobra.Command{Use: "logout", Short: "Forget the managed session and local reference", Args: exact(0), RunE: func(cmd *cobra.Command, _ []string) error { return a.removeProfile(cmd.Context(), a.profile, yes) }}
	logout.Flags().BoolVar(&yes, "yes", false, "confirm removal")
	service := &cobra.Command{Use: "service", Short: "Inspect or stop the managed session engine", Args: exact(0), RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() }}
	service.AddCommand(&cobra.Command{Use: "status", Args: exact(0), RunE: func(cmd *cobra.Command, _ []string) error {
		return a.print(galleton.RuntimeStatus(cmd.Context(), filepath.Join(a.dir, "galleton")))
	}}, &cobra.Command{Use: "stop", Args: exact(0), RunE: func(cmd *cobra.Command, _ []string) error {
		if galleton.Configured() {
			return usage("external engine lifecycle belongs to its owner")
		}
		dir := filepath.Join(a.dir, "galleton")
		if _, err := os.Stat(dir); err == nil {
			if err = galleton.Stop(cmd.Context(), dir); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		return a.print(map[string]bool{"stopped": true})
	}})
	cmd.AddCommand(login, status, refresh, logout, service, &cobra.Command{Use: "export-script", Short: "Print the signed-in browser export script", Args: exact(0), RunE: func(_ *cobra.Command, _ []string) error {
		_, err := fmt.Fprint(a.stdout, scripts.SessionExport)
		return err
	}})
	return cmd
}

// Profile mutations share a lock so concurrent login/logout cannot orphan a session.
func (a *app) lockProfile(ctx context.Context, path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	return filelock.Acquire(ctx, path+".lock")
}

// Keep IDs durable until forgetting succeeds. Retrying deletion is idempotent.
func cleanupRetired(ctx context.Context, path string, session *elevenlabs.Session, engine *galleton.Client) error {
	if len(session.RetiredIDs) == 0 {
		return nil
	}
	for _, id := range session.RetiredIDs {
		if err := engine.Forget(ctx, id); err != nil {
			return elevenlabs.ManagedError(err)
		}
	}
	session.RetiredIDs = nil
	return elevenlabs.SaveSession(path, session)
}
