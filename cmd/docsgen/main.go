package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Microck/elevenlabs-web-cli/internal/cli"
)

func main() {
	check := flag.Bool("check", false, "verify the generated reference without writing it")
	flag.Parse()
	reference := cli.CommandReference()
	files := map[string]string{
		"docs/commands.md":                     reference,
		"docs-site/content/docs/reference.mdx": "---\ntitle: Command reference\ndescription: Generated from the CLI command tree.\n---\n\n" + strings.TrimPrefix(reference, "# Command reference\n\n"),
	}
	for path, expected := range files {
		if *check {
			actual, err := os.ReadFile(path)
			if err != nil || string(actual) != expected {
				fmt.Fprintln(os.Stderr, "command reference is stale; run make docs")
				os.Exit(1)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(path, []byte(expected), 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
