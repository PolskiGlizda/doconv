package converter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	maroto "github.com/johnfercher/maroto/v2"
	marotorow "github.com/johnfercher/maroto/v2/pkg/components/row"
	marototext "github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	goldtext "github.com/yuin/goldmark/text"
)

func init() {
	Register("md", "pdf", ConverterFunc(mdToPDF))
}

// mdToPDF converts Markdown to PDF by routing through the HTML pipeline:
//
//	md → HTML (goldmark, full fidelity) → PDF (Chrome if available, else fallback)
//
// This ensures tables, code blocks, inline formatting, and all GFM features
// are rendered correctly. The direct maroto path (mdToPDFDirect) is kept as
// the base renderer for the pure-Go fallback chain inside html2pdf.go.
func mdToPDF(ctx context.Context, src io.Reader, dst io.Writer) error {
	var htmlBuf bytes.Buffer
	if err := mdToHTML(ctx, src, &htmlBuf); err != nil {
		return fmt.Errorf("md→html: %w", err)
	}
	return htmlToPDF(ctx, &htmlBuf, dst)
}

// ── Pure-Go fallback renderer (used by html2pdf.go when Chrome is absent) ──────

// mdToPDFDirect converts Markdown to PDF using a pure-Go maroto layout engine.
// It is not registered as a public converter; it is called by the html→pdf
// fallback chain to avoid a circular call through mdToPDF.
//
// Quality is limited: inline formatting and tables are rendered as plain text,
// but the output is readable and requires no external binaries.
func mdToPDFDirect(ctx context.Context, src io.Reader, dst io.Writer) error {
	input, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("read markdown: %w", err)
	}

	cfg := config.NewBuilder().WithPageNumber().Build()
	mrt := maroto.New(cfg)

	reader := goldtext.NewReader(input)
	parser := goldmark.DefaultParser()
	doc := parser.Parse(reader)

	var rows []core.Row
	walkErr := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Heading:
			content := extractText(node, input)
			rows = append(rows, marotorow.New(10).Add(
				marototext.NewCol(12, content, headingProps(node.Level)),
			))
			return ast.WalkSkipChildren, nil
		case *ast.Paragraph:
			content := extractText(node, input)
			if strings.TrimSpace(content) == "" {
				return ast.WalkSkipChildren, nil
			}
			rows = append(rows, marotorow.New(8).Add(
				marototext.NewCol(12, content, props.Text{Size: 10, Align: align.Left}),
			))
			return ast.WalkSkipChildren, nil
		case *ast.CodeBlock, *ast.FencedCodeBlock:
			content := extractText(node, input)
			rows = append(rows, marotorow.New(8).Add(
				marototext.NewCol(12, content, props.Text{Size: 9, Style: fontstyle.Italic, Align: align.Left}),
			))
			return ast.WalkSkipChildren, nil
		case *ast.Blockquote:
			content := "> " + extractText(node, input)
			rows = append(rows, marotorow.New(8).Add(
				marototext.NewCol(12, content, props.Text{Size: 10, Style: fontstyle.Italic, Align: align.Left}),
			))
			return ast.WalkSkipChildren, nil
		case *ast.ListItem:
			content := "• " + strings.TrimSpace(extractText(node, input))
			rows = append(rows, marotorow.New(7).Add(
				marototext.NewCol(12, content, props.Text{Size: 10, Align: align.Left}),
			))
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	if walkErr != nil {
		return fmt.Errorf("walk ast: %w", walkErr)
	}

	mrt.AddRows(rows...)

	pdfdoc, genErr := mrt.Generate()
	if genErr != nil {
		return fmt.Errorf("generate pdf: %w", genErr)
	}
	_, err = io.Copy(dst, bytes.NewReader(pdfdoc.GetBytes()))
	return err
}

// headingProps maps AST heading level (1–6) to maroto text props.
func headingProps(level int) props.Text {
	sizes := []float64{18, 15, 13, 12, 11, 10}
	sz := sizes[0]
	if level >= 1 && level <= 6 {
		sz = sizes[level-1]
	}
	return props.Text{Size: sz, Style: fontstyle.Bold, Align: align.Left}
}

// extractText returns the plain text content of an AST node.
// Block nodes expose source Lines; inline nodes expose a Segment.
// Falls back to walking children for composite nodes.
// Strips any residual markdown syntax characters.
func extractText(n ast.Node, src []byte) string {
	var b strings.Builder
	extractTextInto(n, src, &b)
	result := strings.TrimRight(b.String(), "\n")
	return reMarkdownSyntax.ReplaceAllString(result, "$1")
}

// reMarkdownSyntax strips leading markdown emphasis markers so that
// **bold**, *italic*, `code` don't leak into plain-text output.
var reMarkdownSyntax = regexp.MustCompile(`\*{1,3}([^*]+)\*{1,3}|` + "`([^`]+)`")

func extractTextInto(n ast.Node, src []byte, b *strings.Builder) {
	// Inline text leaf (most common inline node)
	if t, ok := n.(*ast.Text); ok {
		b.Write(t.Segment.Value(src))
		return
	}
	// Block nodes carry source lines
	if n.Type() == ast.TypeBlock {
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			b.Write(seg.Value(src))
		}
		if lines.Len() > 0 {
			return
		}
	}
	// Composite / inline nodes without a direct segment: walk children
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		extractTextInto(c, src, b)
	}
}
