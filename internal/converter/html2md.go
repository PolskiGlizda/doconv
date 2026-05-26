package converter

import (
	"context"
	"fmt"
	"io"

	htmltomd "github.com/JohannesKaufmann/html-to-markdown/v2"
)

func init() {
	Register("html", "md", ConverterFunc(htmlToMD))
}

func htmlToMD(_ context.Context, src io.Reader, dst io.Writer) error {
	input, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("read html: %w", err)
	}
	md, err := htmltomd.ConvertString(string(input))
	if err != nil {
		return fmt.Errorf("convert html to markdown: %w", err)
	}
	_, err = fmt.Fprint(dst, md)
	return err
}
