package converter

import (
	"context"
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/fumiama/go-docx"
)

func init() {
	Register("docx", "html", ConverterFunc(docxToHTML))
}

func docxToHTML(_ context.Context, src io.Reader, dst io.Writer) error {
	data, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("read docx: %w", err)
	}

	doc, err := docx.Parse(newReaderAt(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("parse docx: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html>\n<head><meta charset=\"utf-8\"></head>\n<body>\n")

	for _, item := range doc.Document.Body.Items {
		switch t := item.(type) {
		case *docx.Paragraph:
			sb.WriteString(paraToHTML(t))
		case *docx.Table:
			sb.WriteString(wtableToHTML(t))
		}
	}

	sb.WriteString("</body>\n</html>\n")
	_, err = fmt.Fprint(dst, sb.String())
	return err
}

func paraToHTML(p *docx.Paragraph) string {
	if p == nil {
		return ""
	}

	// Determine heading level from style
	tag := ""
	if p.Properties != nil && p.Properties.Style != nil {
		tag = styleToHTMLTag(p.Properties.Style.Val)
	}

	content := paraRichHTML(p)
	if strings.TrimSpace(content) == "" {
		return "<br>\n"
	}
	if tag != "" {
		return "<" + tag + ">" + content + "</" + tag + ">\n"
	}
	return "<p>" + content + "</p>\n"
}

// paraRichHTML renders the paragraph children with inline bold/italic markup.
func paraRichHTML(p *docx.Paragraph) string {
	var sb strings.Builder
	for _, child := range p.Children {
		switch r := child.(type) {
		case *docx.Run:
			txt := html.EscapeString(runPlainText(r))
			bold := r.RunProperties != nil && r.RunProperties.Bold != nil
			italic := r.RunProperties != nil && r.RunProperties.Italic != nil
			if bold && italic {
				txt = "<strong><em>" + txt + "</em></strong>"
			} else if bold {
				txt = "<strong>" + txt + "</strong>"
			} else if italic {
				txt = "<em>" + txt + "</em>"
			}
			sb.WriteString(txt)
		case *docx.Hyperlink:
			sb.WriteString(html.EscapeString(r.Run.InstrText))
		}
	}
	return sb.String()
}

// runPlainText collects text from all *Text children of a Run.
func runPlainText(r *docx.Run) string {
	var sb strings.Builder
	for _, c := range r.Children {
		if t, ok := c.(*docx.Text); ok {
			sb.WriteString(t.Text)
		}
	}
	return sb.String()
}

func styleToHTMLTag(style string) string {
	switch style {
	case "Heading1", "heading1", "1":
		return "h1"
	case "Heading2", "heading2", "2":
		return "h2"
	case "Heading3", "heading3", "3":
		return "h3"
	case "Heading4", "heading4", "4":
		return "h4"
	case "Heading5", "heading5", "5":
		return "h5"
	case "Heading6", "heading6", "6":
		return "h6"
	}
	return ""
}

func wtableToHTML(t *docx.Table) string {
	var sb strings.Builder
	sb.WriteString("<table>\n")
	for _, row := range t.TableRows {
		sb.WriteString("<tr>")
		for _, cell := range row.TableCells {
			sb.WriteString("<td>")
			for _, p := range cell.Paragraphs {
				sb.WriteString(html.EscapeString(p.String()))
			}
			sb.WriteString("</td>")
		}
		sb.WriteString("</tr>\n")
	}
	sb.WriteString("</table>\n")
	return sb.String()
}
