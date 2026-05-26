package cli

import (
	"fmt"
	"strings"

	"github.com/PolskiGlizda/doconv/internal/converter"
	"github.com/spf13/cobra"
)

// FormatsCmd returns the `formats` subcommand.
func FormatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "formats",
		Short: "List all supported conversion routes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			routes := converter.Routes()
			if len(routes) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No converters registered.")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Supported conversions:")
			for _, r := range routes {
				parts := strings.SplitN(r, "→", 2)
				if len(parts) == 2 {
					fmt.Fprintf(cmd.OutOrStdout(), "  %-8s →  %s\n", parts[0], parts[1])
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", r)
				}
			}
			return nil
		},
	}
}
