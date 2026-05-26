package converter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bmaupin/go-epub"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

func init() {
	Register("md", "epub", ConverterFunc(mdToEPUB))
}

var gmEPUB = goldmark.New(
	goldmark.WithExtensions(extension.GFM, extension.Footnote),
)

func mdToEPUB(_ context.Context, src io.Reader, dst io.Writer) error {
	input, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("read markdown: %w", err)
	}

	// Render markdown → HTML fragment
	var htmlBuf bytes.Buffer
	if err := gmEPUB.Convert(input, &htmlBuf); err != nil {
		return fmt.Errorf("render html: %w", err)
	}

	book := epub.NewEpub("Document")
	book.SetAuthor("")

	_, err = book.AddSection(htmlBuf.String(), "Content", "", "")
	if err != nil {
		return fmt.Errorf("add epub section: %w", err)
	}

	// go-epub only writes to a file path, so use a temp file then copy.
	tmp, err := os.CreateTemp("", "doconv-*.epub")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmp.Close()
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := book.Write(filepath.Clean(tmpPath)); err != nil {
		return fmt.Errorf("write epub: %w", err)
	}

	f, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("open temp epub: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(dst, f)
	return err
}
