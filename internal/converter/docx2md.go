package converter

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/fumiama/go-docx"
)

func init() {
	Register("docx", "md", ConverterFunc(docxToMD))
}

func docxToMD(_ context.Context, src io.Reader, dst io.Writer) error {
	data, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("read docx: %w", err)
	}

	doc, err := docx.Parse(newReaderAt(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("parse docx: %w", err)
	}

	var sb strings.Builder
	for _, item := range doc.Document.Body.Items {
		switch t := item.(type) {
		case *docx.Paragraph:
			sb.WriteString(paraToMD(t))
		case *docx.Table:
			sb.WriteString(wtableToMD(t))
		}
	}

	_, err = fmt.Fprint(dst, sb.String())
	return err
}

func paraToMD(p *docx.Paragraph) string {
	if p == nil {
		return ""
	}

	prefix := ""
	if p.Properties != nil && p.Properties.Style != nil {
		prefix = styleToMDPrefix(p.Properties.Style.Val)
	}

	content := paraRichMD(p)
	text := strings.TrimSpace(content)
	if text == "" {
		return "\n"
	}
	return prefix + text + "\n\n"
}

// paraRichMD renders paragraph children with inline bold/italic Markdown.
func paraRichMD(p *docx.Paragraph) string {
	var sb strings.Builder
	for _, child := range p.Children {
		switch r := child.(type) {
		case *docx.Run:
			txt := runPlainText(r)
			bold := r.RunProperties != nil && r.RunProperties.Bold != nil
			italic := r.RunProperties != nil && r.RunProperties.Italic != nil
			if bold && italic {
				txt = "***" + txt + "***"
			} else if bold {
				txt = "**" + txt + "**"
			} else if italic {
				txt = "_" + txt + "_"
			}
			sb.WriteString(txt)
		case *docx.Hyperlink:
			sb.WriteString(r.Run.InstrText)
		}
	}
	return sb.String()
}

func styleToMDPrefix(style string) string {
	switch style {
	case "Heading1", "heading1", "1":
		return "# "
	case "Heading2", "heading2", "2":
		return "## "
	case "Heading3", "heading3", "3":
		return "### "
	case "Heading4", "heading4", "4":
		return "#### "
	case "Heading5", "heading5", "5":
		return "##### "
	case "Heading6", "heading6", "6":
		return "###### "
	}
	return ""
}

func wtableToMD(t *docx.Table) string {
	if len(t.TableRows) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, row := range t.TableRows {
		cells := make([]string, len(row.TableCells))
		for j, cell := range row.TableCells {
			var cellText strings.Builder
			for _, p := range cell.Paragraphs {
				cellText.WriteString(p.String())
			}
			cells[j] = strings.TrimSpace(cellText.String())
		}
		sb.WriteString("| " + strings.Join(cells, " | ") + " |\n")
		if i == 0 {
			sep := make([]string, len(cells))
			for j := range sep {
				sep[j] = "---"
			}
			sb.WriteString("| " + strings.Join(sep, " | ") + " |\n")
		}
	}
	sb.WriteString("\n")
	return sb.String()
}
