package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/PolskiGlizda/doconv/internal/apperr"
	"github.com/PolskiGlizda/doconv/internal/converter"
	"github.com/PolskiGlizda/doconv/internal/detect"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

// ConvertCmd returns the `convert` subcommand.
func ConvertCmd() *cobra.Command {
	var (
		fromFmt string
		toFmt   string
		verbose bool
	)

	cmd := &cobra.Command{
		Use:   "convert <input> <output>",
		Short: "Convert a document from one format to another",
		Example: `  doconv convert README.md out.pdf
  doconv convert page.html doc.md
  doconv convert report.docx summary.md
  doconv convert doc.md book.epub`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputPath := args[0]
			outputPath := args[1]

			// Resolve source format
			srcFmt, err := resolveFormat(fromFmt, inputPath, "source")
			if err != nil {
				return err
			}

			// Resolve target format
			dstFmt, err := resolveFormat(toFmt, outputPath, "target")
			if err != nil {
				return err
			}

			// Ensure converters are registered (import side-effects)
			_ = converter.Routes()

			conv, err := converter.Get(srcFmt, dstFmt)
			if err != nil {
				return err // UnsupportedFormatError → exit 2 in main
			}

			if verbose {
				fmt.Fprintf(cmd.ErrOrStderr(), "converting %s (%s) → %s (%s)\n",
					inputPath, srcFmt, outputPath, dstFmt)
			}

			// Open input
			in, err := os.Open(inputPath)
			if err != nil {
				return fmt.Errorf("open input: %w", err)
			}
			defer in.Close()

			// Create output file
			out, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("create output: %w", err)
			}
			defer func() {
				out.Close()
				// Remove partially-written output on error (handled below).
			}()

			// Progress spinner (indeterminate — we don't know output size)
			bar := progressbar.NewOptions(-1,
				progressbar.OptionSetWriter(cmd.ErrOrStderr()),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionSetDescription(fmt.Sprintf("converting to %s", dstFmt)),
				progressbar.OptionClearOnFinish(),
				progressbar.OptionShowElapsedTimeOnFinish(),
			)
			defer bar.Close()

			// Run the conversion
			w := io.MultiWriter(out, progressWriter{bar})
			convErr := conv.Convert(context.Background(), in, w)
			bar.Finish()

			if convErr != nil {
				out.Close()
				os.Remove(outputPath) // clean up partial file
				return &apperr.ConversionError{
					SourceFile:   inputPath,
					TargetFormat: dstFmt,
					Cause:        convErr,
				}
			}

			if verbose {
				fi, _ := os.Stat(outputPath)
				if fi != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "wrote %s (%d bytes)\n", outputPath, fi.Size())
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&fromFmt, "from", "", "source format override (md, html, docx, pdf, epub)")
	cmd.Flags().StringVar(&toFmt, "to", "", "target format override (md, html, docx, pdf, epub)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print conversion details to stderr")

	return cmd
}

func resolveFormat(flag, path, label string) (string, error) {
	if flag != "" {
		f, err := detect.Canonical(flag)
		if err != nil {
			return "", fmt.Errorf("unknown %s format %q (use md, html, docx, pdf, or epub): %w", label, flag, err)
		}
		return f, nil
	}
	f, err := detect.FromPath(path)
	if err != nil {
		var ufe *apperr.UnsupportedFormatError
		if errors.As(err, &ufe) {
			flagName := map[string]string{"source": "from", "target": "to"}[label]
			if flagName == "" {
				flagName = "from/to"
			}
			return "", fmt.Errorf("cannot detect %s format from %q (use --%s to specify): %w", label, path, flagName, ufe)
		}
		return "", err
	}
	return f, nil
}

// progressWriter ticks the spinner on every Write so it stays alive during
// slow conversions (e.g. chromedp).
type progressWriter struct{ bar *progressbar.ProgressBar }

func (pw progressWriter) Write(p []byte) (int, error) {
	_ = pw.bar.Add(len(p))
	return len(p), nil
}
