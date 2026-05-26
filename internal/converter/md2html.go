package converter

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

func init() {
	Register("md", "html", ConverterFunc(mdToHTML))
}

var gm = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,        // tables, strikethrough, autolinks, task lists
		extension.Footnote,
		extension.DefinitionList,
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithXHTML(),
	),
)

func mdToHTML(_ context.Context, src io.Reader, dst io.Writer) error {
	input, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("read markdown: %w", err)
	}
	var buf bytes.Buffer
	if err := gm.Convert(input, &buf); err != nil {
		return fmt.Errorf("render html: %w", err)
	}
	// Wrap in a minimal HTML document so the output is valid standalone HTML.
	_, err = fmt.Fprintf(dst,
		"<!DOCTYPE html>\n<html>\n<head><meta charset=\"utf-8\"></head>\n<body>\n%s\n</body>\n</html>\n",
		buf.String(),
	)
	return err
}
