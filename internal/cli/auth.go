package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Microck/elevenlabs-web-cli/internal/elevenlabs"
	"github.com/Microck/elevenlabs-web-cli/scripts"
	"github.com/spf13/cobra"
)

func (a *app) auth() *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Import and manage a browser session"}
	var input string
	login := &cobra.Command{Use: "login", Short: "Import a session exported with scripts/export-session.js", Args: exact(0), RunE: func(_ *cobra.Command, _ []string) error {
		raw, err := readInput(input, a.stdin)
		if err != nil {
			return err
		}
		var session elevenlabs.Session
		if err := json.Unmarshal(raw, &session); err != nil {
			return usage("invalid session JSON; use scripts/export-session.js in the browser")
		}
		path, err := elevenlabs.SessionPath(a.dir, a.profile)
		if err != nil {
			return err
		}
		if err := elevenlabs.SaveSession(path, &session); err != nil {
			return err
		}
		return a.print(map[string]any{"profile": a.profile, "imported": true, "expires_at": session.ExpiresAt})
	}}
	login.Flags().StringVar(&input, "session", "", "session JSON file, or - for stdin")
	status := &cobra.Command{Use: "status", Short: "Show local session presence and expiry", Args: exact(0), RunE: func(_ *cobra.Command, _ []string) error {
		if err := a.connect(); err != nil {
			return err
		}
		return a.print(map[string]any{"profile": a.profile, "expires_at": a.client.Session.ExpiresAt, "expired": time.Now().After(a.client.Session.ExpiresAt)})
	}}
	refresh := &cobra.Command{Use: "refresh", Short: "Renew the session now", Args: exact(0), RunE: func(cmd *cobra.Command, _ []string) error {
		if err := a.connect(); err != nil {
			return err
		}
		if err := a.client.Refresh(cmd.Context()); err != nil {
			return err
		}
		return a.print(map[string]any{"profile": a.profile, "expires_at": a.client.Session.ExpiresAt})
	}}
	var yes bool
	logout := &cobra.Command{Use: "logout", Short: "Remove the local profile session", Args: exact(0), RunE: func(_ *cobra.Command, _ []string) error {
		return a.removeProfile(a.profile, yes)
	}}
	logout.Flags().BoolVar(&yes, "yes", false, "confirm removal")
	cmd.AddCommand(login, status, refresh, logout, &cobra.Command{Use: "export-script", Short: "Print the script to export a signed-in browser session", Args: exact(0), RunE: func(_ *cobra.Command, _ []string) error {
		_, err := fmt.Fprint(a.stdout, scripts.SessionExport)
		return err
	}})
	return cmd
}
