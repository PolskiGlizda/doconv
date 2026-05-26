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
		extension.GFM,       // tables, strikethrough, autolinks, task lists
		extension.Footnote,
		extension.DefinitionList,
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithXHTML(),
	),
)

// htmlCSS is a GitHub-flavoured stylesheet embedded in every HTML output.
// It makes standalone HTML readable and prints well when Chrome renders it to PDF.
const htmlCSS = `
<style>
* { box-sizing: border-box; }
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  font-size: 16px;
  line-height: 1.6;
  max-width: 900px;
  margin: 40px auto;
  padding: 0 24px;
  color: #24292e;
}
h1, h2, h3, h4, h5, h6 {
  margin-top: 24px;
  margin-bottom: 16px;
  font-weight: 600;
  line-height: 1.25;
}
h1 { font-size: 2em;    border-bottom: 1px solid #eaecef; padding-bottom: .3em; }
h2 { font-size: 1.5em;  border-bottom: 1px solid #eaecef; padding-bottom: .3em; }
h3 { font-size: 1.25em; }
h4 { font-size: 1em; }
p  { margin-top: 0; margin-bottom: 16px; }
a  { color: #0366d6; text-decoration: none; }
a:hover { text-decoration: underline; }
code {
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
  font-size: 85%;
  background: rgba(27,31,35,.05);
  padding: .2em .4em;
  border-radius: 3px;
}
pre {
  background: #f6f8fa;
  border-radius: 6px;
  padding: 16px;
  overflow: auto;
  font-size: 85%;
  line-height: 1.45;
  margin-bottom: 16px;
}
pre code { background: transparent; padding: 0; font-size: 100%; }
blockquote {
  border-left: 4px solid #dfe2e5;
  padding: 0 1em;
  color: #6a737d;
  margin: 0 0 16px 0;
}
table {
  border-collapse: collapse;
  width: 100%;
  margin-bottom: 16px;
  display: block;
  overflow-x: auto;
}
th, td {
  border: 1px solid #dfe2e5;
  padding: 6px 13px;
  text-align: left;
}
th { background: #f6f8fa; font-weight: 600; }
tr:nth-child(even) td { background: #f6f8fa; }
ul, ol { padding-left: 2em; margin-bottom: 16px; }
li { margin-bottom: 4px; }
hr { border: 0; border-top: 1px solid #eaecef; margin: 24px 0; }
img { max-width: 100%; }
@media print {
  body { max-width: 100%; margin: 0; padding: 16px; font-size: 14px; }
  pre  { white-space: pre-wrap; }
  a    { color: inherit; }
}
</style>
`

func mdToHTML(_ context.Context, src io.Reader, dst io.Writer) error {
	input, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("read markdown: %w", err)
	}
	var buf bytes.Buffer
	if err := gm.Convert(input, &buf); err != nil {
		return fmt.Errorf("render html: %w", err)
	}
	_, err = fmt.Fprintf(dst,
		"<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n%s\n</head>\n<body>\n%s\n</body>\n</html>\n",
		htmlCSS,
		buf.String(),
	)
	return err
}
