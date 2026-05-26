package converter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func init() {
	Register("html", "pdf", ConverterFunc(htmlToPDF))
}

// chromeBinaries is the ordered list of Chrome/Chromium executable names to
// probe at runtime.
var chromeBinaries = []string{
	"google-chrome",
	"google-chrome-stable",
	"chromium",
	"chromium-browser",
	"microsoft-edge",
	"msedge",
}

// findChrome returns the path to a Chrome/Chromium binary, or "" if none found.
func findChrome() string {
	for _, name := range chromeBinaries {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

func htmlToPDF(ctx context.Context, src io.Reader, dst io.Writer) error {
	input, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("read html: %w", err)
	}

	if path := findChrome(); path != "" {
		return htmlToPDFChrome(ctx, input, dst, path)
	}

	// Fallback: direct HTML → PDF without Chrome.
	// Uses htmlToPDFFallback which parses the HTML tree and renders it to
	// maroto, preserving tables, headings, code blocks, and inline formatting.
	return htmlToPDFFallback(ctx, input, dst)
}

func htmlToPDFChrome(ctx context.Context, htmlInput []byte, dst io.Writer, chromePath string) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	// Encode the HTML as a data URL to avoid writing a temp file.
	dataURL := "data:text/html;charset=utf-8," + percentEncode(string(htmlInput))

	var pdfBuf []byte
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(dataURL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				Do(ctx)
			return err
		}),
	)
	if err != nil {
		return fmt.Errorf("chromedp print to pdf: %w", err)
	}

	_, err = io.Copy(dst, bytes.NewReader(pdfBuf))
	return err
}

// percentEncode performs minimal percent-encoding for a data: URI payload.
// Characters that must be encoded are control characters and the few
// delimiters that confuse browsers inside data URIs.
func percentEncode(s string) string {
	var b bytes.Buffer
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '%':
			fmt.Fprintf(&b, "%%25")
		case c == '#':
			fmt.Fprintf(&b, "%%23")
		case c < 0x20 && c != '\t' && c != '\n' && c != '\r':
			fmt.Fprintf(&b, "%%%02X", c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
