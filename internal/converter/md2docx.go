package converter

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/fumiama/go-docx"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	goldtext "github.com/yuin/goldmark/text"
)

func init() {
	Register("md", "docx", ConverterFunc(mdToDOCX))
}

func mdToDOCX(_ context.Context, src io.Reader, dst io.Writer) error {
	input, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("read markdown: %w", err)
	}

	reader := goldtext.NewReader(input)
	parser := goldmark.DefaultParser()
	doc := parser.Parse(reader)

	w := docx.New().WithDefaultTheme()

	walkErr := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Heading:
			content := extractText(node, input)
			style := fmt.Sprintf("Heading%d", node.Level)
			w.AddParagraph().Style(style).AddText(content)
			return ast.WalkSkipChildren, nil

		case *ast.Paragraph:
			content := extractText(node, input)
			if strings.TrimSpace(content) == "" {
				return ast.WalkSkipChildren, nil
			}
			// Walk inline children for bold/italic spans
			addStyledParagraph(w, node, input)
			return ast.WalkSkipChildren, nil

		case *ast.CodeBlock, *ast.FencedCodeBlock:
			content := extractText(node, input)
			w.AddParagraph().AddText(content).Font("Courier New", "", "Courier New", "")
			return ast.WalkSkipChildren, nil

		case *ast.ListItem:
			content := strings.TrimSpace(extractText(node, input))
			w.AddParagraph().AddText("• " + content)
			return ast.WalkSkipChildren, nil

		case *ast.Blockquote:
			content := extractText(node, input)
			w.AddParagraph().AddText("> " + content)
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	if walkErr != nil {
		return fmt.Errorf("walk ast: %w", walkErr)
	}

	_, err = w.WriteTo(dst)
	return err
}

// addStyledParagraph creates a paragraph from an AST Paragraph node,
// applying bold/italic to inline Emphasis spans.
func addStyledParagraph(w *docx.Docx, node *ast.Paragraph, src []byte) {
	para := w.AddParagraph()
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		switch inline := c.(type) {
		case *ast.Emphasis:
			text := extractInlineText(inline, src)
			r := para.AddText(text)
			if inline.Level == 2 { // ** = bold
				r.Bold()
			} else { // * = italic
				r.Italic()
			}
		case *ast.Text:
			para.AddText(string(inline.Segment.Value(src)))
		default:
			para.AddText(extractInlineText(c, src))
		}
	}
}

// extractInlineText extracts plain text from an inline AST node by walking
// its *ast.Text leaf children.
func extractInlineText(n ast.Node, src []byte) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			b.Write(t.Segment.Value(src))
		} else {
			b.WriteString(extractInlineText(c, src))
		}
	}
	return b.String()
}
