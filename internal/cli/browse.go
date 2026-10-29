package cli

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func (a *app) voiceSearch() *cobra.Command {
	var pageSize, page int
	var language, sort string
	cmd := &cobra.Command{Use: "search TEXT", Short: "Search the shared voice library", Args: exact(1), RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(args[0]) == "" {
			return usage("search text cannot be empty")
		}
		if err := checkPageSize(pageSize); err != nil {
			return err
		}
		if page < 0 {
			return usage("--page cannot be negative")
		}
		query := url.Values{"search": {args[0]}, "sort": {sort}, "page_size": {strconv.Itoa(pageSize)}, "page": {strconv.Itoa(page)}, "explore_source": {"voice_library_creative"}}
		if language != "" {
			query.Set("language", language)
		}
		return a.queryPage(cmd, "/v1/shared-voices", query)
	}}
	cmd.Flags().IntVar(&pageSize, "page-size", 30, "voices per page (1-100)")
	cmd.Flags().IntVar(&page, "page", 0, "zero-based page number")
	cmd.Flags().StringVar(&language, "language", "", "language code, such as en or es")
	cmd.Flags().StringVar(&sort, "sort", "trending", "voice library sort order")
	return cmd
}

func (a *app) history() *cobra.Command {
	var pageSize int
	var source, search string
	parent := &cobra.Command{Use: "history", Short: "Inspect generation history"}
	cmd := &cobra.Command{Use: "list", Short: "List the first page of generation history", Args: exact(0), RunE: func(cmd *cobra.Command, _ []string) error {
		if err := checkPageSize(pageSize); err != nil {
			return err
		}
		query := url.Values{"page_size": {strconv.Itoa(pageSize)}, "sort_direction": {"desc"}}
		if source != "" {
			query.Set("source", source)
		}
		if search != "" {
			query.Set("search", search)
		}
		return a.queryPage(cmd, "/v1/history", query)
	}}
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "history items per page (1-100)")
	cmd.Flags().StringVar(&source, "source", "", "filter by source, such as TTS")
	cmd.Flags().StringVar(&search, "search", "", "search history text")
	parent.AddCommand(cmd, a.resourceRead("show ID", "Show history details", "/v1/history/", ""), a.historyDownload())
	return parent
}

func checkPageSize(size int) error {
	if size < 1 || size > 100 {
		return usage("--page-size must be between 1 and 100")
	}
	return nil
}

// Keep page envelopes intact: scripts need has_more and continuation metadata.
func (a *app) queryPage(cmd *cobra.Command, path string, query url.Values) error {
	if err := a.connect(); err != nil {
		return err
	}
	value, err := a.client.JSON(cmd.Context(), "GET", path+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	return a.print(value)
}

func (a *app) resourceRead(use, short, prefix, suffix string) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Args: exact(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkID(args[0]); err != nil {
			return err
		}
		if err := a.connect(); err != nil {
			return err
		}
		value, err := a.client.JSON(cmd.Context(), "GET", prefix+args[0]+suffix, nil)
		if err != nil {
			return err
		}
		return a.print(value)
	}}
}
