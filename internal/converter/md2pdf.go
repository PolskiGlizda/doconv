package converter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	maroto "github.com/johnfercher/maroto/v2"
	marototext "github.com/johnfercher/maroto/v2/pkg/components/text"
	marotorow "github.com/johnfercher/maroto/v2/pkg/components/row"
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

// headingProps maps AST heading level (1–6) to maroto text props.
func headingProps(level int) props.Text {
	sizes := []float64{18, 15, 13, 12, 11, 10}
	sz := sizes[0]
	if level >= 1 && level <= 6 {
		sz = sizes[level-1]
	}
	return props.Text{Size: sz, Style: fontstyle.Bold, Align: align.Left}
}

func mdToPDF(_ context.Context, src io.Reader, dst io.Writer) error {
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
			rows = append(rows, marotorow.New(10).Add(marototext.NewCol(12, content, headingProps(node.Level))))
			return ast.WalkSkipChildren, nil
		case *ast.Paragraph:
			content := extractText(node, input)
			rows = append(rows, marotorow.New(8).Add(marototext.NewCol(12, content, props.Text{Size: 10, Align: align.Left})))
			return ast.WalkSkipChildren, nil
		case *ast.CodeBlock, *ast.FencedCodeBlock:
			content := extractText(node, input)
			rows = append(rows, marotorow.New(8).Add(marototext.NewCol(12, content, props.Text{Size: 9, Style: fontstyle.Italic, Align: align.Left})))
			return ast.WalkSkipChildren, nil
		case *ast.Blockquote:
			content := "> " + extractText(node, input)
			rows = append(rows, marotorow.New(8).Add(marototext.NewCol(12, content, props.Text{Size: 10, Style: fontstyle.Italic, Align: align.Left})))
			return ast.WalkSkipChildren, nil
		case *ast.ListItem:
			content := "• " + strings.TrimSpace(extractText(node, input))
			rows = append(rows, marotorow.New(7).Add(marototext.NewCol(12, content, props.Text{Size: 10, Align: align.Left})))
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

// extractText returns the plain text content of an AST node.
// Block nodes expose source Lines; inline nodes expose a Segment.
// Falls back to walking children for composite nodes.
func extractText(n ast.Node, src []byte) string {
	var b strings.Builder
	extractTextInto(n, src, &b)
	return strings.TrimRight(b.String(), "\n")
}

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
