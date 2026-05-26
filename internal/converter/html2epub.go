package converter

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bmaupin/go-epub"
)

func init() {
	Register("html", "epub", ConverterFunc(htmlToEPUB))
}

func htmlToEPUB(_ context.Context, src io.Reader, dst io.Writer) error {
	input, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("read html: %w", err)
	}

	book := epub.NewEpub("Document")
	_, err = book.AddSection(string(input), "Content", "", "")
	if err != nil {
		return fmt.Errorf("add epub section: %w", err)
	}

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
