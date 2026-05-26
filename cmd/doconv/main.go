package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/PolskiGlizda/doconv/internal/apperr"
	"github.com/PolskiGlizda/doconv/internal/cli"

	// Import converters for their init() side-effects (registrations).
	_ "github.com/PolskiGlizda/doconv/internal/converter"

	"github.com/spf13/cobra"
)

var version = "dev" // overridden at build time via -ldflags

func main() {
	root := &cobra.Command{
		Use:   "doconv",
		Short: "doconv — convert documents between formats",
		Long: `doconv converts documents between Markdown, HTML, PDF, DOCX, and EPUB.

No external binaries are required. When Google Chrome or Chromium is installed,
HTML → PDF uses it for higher-fidelity output; otherwise a pure-Go fallback is used.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		cli.ConvertCmd(),
		cli.FormatsCmd(),
		versionCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)

		var ufe *apperr.UnsupportedFormatError
		if errors.As(err, &ufe) {
			os.Exit(2) // config/input error
		}
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the doconv version",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "doconv %s\n", version)
		},
	}
}
