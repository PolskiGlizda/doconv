package converter

// htmlToPDFRender is a pure-Go HTML → PDF renderer used as the fallback when
// Chrome is not available. It parses the HTML tree with golang.org/x/net/html
// and maps each block element to a maroto row, preserving headings, paragraphs,
// code blocks, lists, blockquotes, and tables.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	maroto "github.com/johnfercher/maroto/v2"
	marotorow "github.com/johnfercher/maroto/v2/pkg/components/row"
	marototext "github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"golang.org/x/net/html"
)

// htmlToPDFFallback is a direct HTML → PDF renderer that does not require Chrome.
// It replaces the html→md→pdf chain and preserves tables, bold, italic, and code.
func htmlToPDFFallback(_ context.Context, htmlInput []byte, dst io.Writer) error {
	doc, err := html.Parse(bytes.NewReader(htmlInput))
	if err != nil {
		return fmt.Errorf("parse html: %w", err)
	}

	r := &htmlPDFRenderer{}
	r.walkNode(doc, inlineState{})

	cfg := config.NewBuilder().WithPageNumber().Build()
	mrt := maroto.New(cfg)
	mrt.AddRows(r.rows...)

	pdfdoc, genErr := mrt.Generate()
	if genErr != nil {
		return fmt.Errorf("generate pdf: %w", genErr)
	}
	_, err = io.Copy(dst, bytes.NewReader(pdfdoc.GetBytes()))
	return err
}

// ── Line-height estimator ────────────────────────────────────────────────────

// softWrap preprocesses cell text to insert break opportunities in long
// unbreakable tokens.  For each space-separated word longer than 14 characters
// it inserts a space after '/' and '_' so that maroto/fpdf's word-wrapper can
// break file paths and identifiers within narrow columns.
func softWrap(s string) string {
	if s == "" {
		return s
	}
	words := strings.Fields(s)
	changed := false
	for i, w := range words {
		if len([]rune(w)) <= 14 {
			continue
		}
		var b strings.Builder
		b.Grow(len(w) + 6)
		runes := []rune(w)
		for j, c := range runes {
			b.WriteRune(c)
			if j < len(runes)-1 && (c == '/' || c == '_') {
				b.WriteByte(' ')
			}
		}
		words[i] = b.String()
		changed = true
	}
	if !changed {
		return s
	}
	return strings.Join(words, " ")
}

// estimateLines returns the approximate number of rendered lines for text in a
// maroto column of colSpan grid units (1–12) at the given font size (pt).
//
// Heuristic baseline: Helvetica at 10 pt fits ~80 characters across a full
// 12-unit column on an A4 page with default 10 mm margins.  Scales linearly
// with colSpan and inversely with fontSize.
func estimateLines(text string, colSpan int, fontSize float64) int {
	if colSpan <= 0 {
		colSpan = 1
	}
	baseChars := int(80.0 * (10.0 / fontSize))
	charsPerLine := baseChars * colSpan / 12
	if charsPerLine < 4 {
		charsPerLine = 4
	}
	n := len([]rune(strings.TrimSpace(text)))
	if n == 0 {
		return 1
	}
	lines := (n + charsPerLine - 1) / charsPerLine
	if lines < 1 {
		lines = 1
	}
	return lines
}

// rowHeight converts a line count to a maroto row height in mm.
// Each line is ~5.5 mm tall; 2 mm padding is added above and below.
func rowHeight(lines int, minHeight float64) float64 {
	h := float64(lines)*5.5 + 2.0
	if h < minHeight {
		return minHeight
	}
	return h
}

// ── Inline state ────────────────────────────────────────────────────────────

type inlineState struct {
	bold   bool
	italic bool
	code   bool
}

func (s inlineState) withBold() inlineState   { s.bold = true; return s }
func (s inlineState) withItalic() inlineState { s.italic = true; return s }
func (s inlineState) withCode() inlineState   { s.code = true; return s }

func (s inlineState) toProps(size float64) props.Text {
	p := props.Text{Size: size, Align: align.Left}
	switch {
	case s.bold && s.italic:
		p.Style = fontstyle.BoldItalic
	case s.bold:
		p.Style = fontstyle.Bold
	case s.italic:
		p.Style = fontstyle.Italic
	}
	return p
}

// ── Renderer ────────────────────────────────────────────────────────────────

type htmlPDFRenderer struct {
	rows []core.Row
}

func (r *htmlPDFRenderer) addRow(height float64, col core.Col) {
	r.rows = append(r.rows, marotorow.New(height).Add(col))
}

// collectText extracts all visible text from a subtree, applying inline styles.
// Returns a flat string suitable for a single maroto cell.
func collectText(n *html.Node, s inlineState) string {
	var b strings.Builder
	collectTextInto(n, s, &b)
	return strings.TrimSpace(b.String())
}

func collectTextInto(n *html.Node, s inlineState, b *strings.Builder) {
	if n.Type == html.TextNode {
		b.WriteString(n.Data)
		return
	}
	if n.Type != html.ElementNode {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			collectTextInto(c, s, b)
		}
		return
	}
	switch n.Data {
	case "strong", "b":
		s = s.withBold()
	case "em", "i":
		s = s.withItalic()
	case "code":
		s = s.withCode()
	case "br":
		b.WriteString(" ")
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectTextInto(c, s, b)
	}
}

// walkNode dispatches block-level elements to dedicated handlers.
func (r *htmlPDFRenderer) walkNode(n *html.Node, s inlineState) {
	if n.Type == html.TextNode {
		txt := strings.TrimSpace(n.Data)
		if txt != "" {
			ht := rowHeight(estimateLines(txt, 12, 10), 7)
			r.addRow(ht, marototext.NewCol(12, txt, s.toProps(10)))
		}
		return
	}
	if n.Type != html.ElementNode {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			r.walkNode(c, s)
		}
		return
	}

	switch n.Data {
	case "h1":
		r.renderHeading(n, 1)
	case "h2":
		r.renderHeading(n, 2)
	case "h3":
		r.renderHeading(n, 3)
	case "h4":
		r.renderHeading(n, 4)
	case "h5":
		r.renderHeading(n, 5)
	case "h6":
		r.renderHeading(n, 6)
	case "p":
		r.renderParagraph(n, s)
	case "pre":
		r.renderPre(n)
	case "code":
		// inline code outside a pre — treat as a small paragraph
		txt := collectText(n, inlineState{})
		if txt != "" {
			ht := rowHeight(estimateLines(txt, 12, 9), 7)
			r.addRow(ht, marototext.NewCol(12, txt, props.Text{Size: 9, Style: fontstyle.Italic, Align: align.Left}))
		}
	case "table":
		r.renderTable(n)
	case "ul", "ol":
		r.renderList(n, n.Data == "ol")
	case "blockquote":
		r.renderBlockquote(n)
	case "hr":
		r.addRow(4, marototext.NewCol(12, strings.Repeat("─", 80), props.Text{Size: 7, Align: align.Left}))
	case "head", "style", "script":
		// skip non-content nodes
	default:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			r.walkNode(c, s)
		}
	}
}

// ── Block renderers ─────────────────────────────────────────────────────────

func (r *htmlPDFRenderer) renderHeading(n *html.Node, level int) {
	txt := collectText(n, inlineState{})
	if txt == "" {
		return
	}
	sizes := []float64{20, 16, 13, 12, 11, 10}
	sz := sizes[0]
	if level >= 1 && level <= 6 {
		sz = sizes[level-1]
	}
	minHeights := []float64{14, 12, 10, 9, 9, 8}
	minHt := minHeights[level-1]
	ht := rowHeight(estimateLines(txt, 12, sz), minHt)
	r.addRow(ht, marototext.NewCol(12, txt, props.Text{
		Size:  sz,
		Style: fontstyle.Bold,
		Align: align.Left,
	}))
}

func (r *htmlPDFRenderer) renderParagraph(n *html.Node, s inlineState) {
	txt := collectText(n, s)
	if txt == "" {
		return
	}
	lines := estimateLines(txt, 12, 10)
	ht := rowHeight(lines, 8)
	r.addRow(ht, marototext.NewCol(12, txt, s.toProps(10)))
}

func (r *htmlPDFRenderer) renderPre(n *html.Node) {
	var b strings.Builder
	collectTextInto(n, inlineState{}, &b)
	text := b.String()
	// Render each line of a code block as a separate small row.
	for _, line := range strings.Split(text, "\n") {
		r.addRow(6, marototext.NewCol(12, line, props.Text{
			Size:  8,
			Style: fontstyle.Italic,
			Align: align.Left,
		}))
	}
}

func (r *htmlPDFRenderer) renderList(n *html.Node, ordered bool) {
	idx := 1
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.Data != "li" {
			continue
		}
		txt := collectText(c, inlineState{})
		if txt == "" {
			idx++
			continue
		}
		bullet := "•  "
		if ordered {
			bullet = fmt.Sprintf("%d.  ", idx)
		}
		full := bullet + txt
		ht := rowHeight(estimateLines(full, 12, 10), 7)
		r.addRow(ht, marototext.NewCol(12, full, props.Text{Size: 10, Align: align.Left}))
		idx++
	}
}

func (r *htmlPDFRenderer) renderBlockquote(n *html.Node) {
	txt := collectText(n, inlineState{})
	if txt == "" {
		return
	}
	full := "│  " + txt
	ht := rowHeight(estimateLines(full, 12, 10), 8)
	r.addRow(ht, marototext.NewCol(12, full, props.Text{
		Size:  10,
		Style: fontstyle.Italic,
		Align: align.Left,
	}))
}

// renderTable renders an HTML table by collecting rows and emitting one maroto
// row per TR.  Column widths are distributed proportionally to the maximum cell
// content length in each column so that wide columns get more space.  Row
// heights grow automatically to accommodate the longest cell in each row.
func (r *htmlPDFRenderer) renderTable(n *html.Node) {
	type tableRow struct {
		cells    []string
		isHeader bool
	}

	var trows []tableRow
	var walkTable func(*html.Node)
	walkTable = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			var cells []string
			isHdr := false
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type != html.ElementNode {
					continue
				}
				if c.Data == "th" || c.Data == "td" {
					if c.Data == "th" {
						isHdr = true
					}
					cells = append(cells, collectText(c, inlineState{}))
				}
			}
			if len(cells) > 0 {
				trows = append(trows, tableRow{cells: cells, isHeader: isHdr})
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkTable(c)
		}
	}
	walkTable(n)

	if len(trows) == 0 {
		return
	}

	// Determine column count.
	cols := 0
	for _, tr := range trows {
		if len(tr.cells) > cols {
			cols = len(tr.cells)
		}
	}
	if cols == 0 {
		return
	}

	// Apply soft-wrapping to all cells so that path-like strings (no spaces)
	// gain break opportunities at '/' boundaries.  Do this before measuring
	// so the length heuristic reflects the same text that will be rendered.
	for i := range trows {
		for j := range trows[i].cells {
			trows[i].cells[j] = softWrap(trows[i].cells[j])
		}
	}

	// Measure the maximum content length (in runes) for each column across all
	// rows. A minimum of 3 runes ensures empty or single-char columns get space.
	maxLen := make([]int, cols)
	for i := range maxLen {
		maxLen[i] = 3
	}
	for _, tr := range trows {
		for i, cell := range tr.cells {
			if i < cols {
				if l := len([]rune(cell)); l > maxLen[i] {
					maxLen[i] = l
				}
			}
		}
	}

	// Convert content lengths to 12-unit maroto grid column widths.
	//
	// Strategy:
	//  1. Cap each column's effective length at 50 chars so that one very long
	//     description column cannot crowd out shorter but important columns.
	//  2. Give every column a guaranteed base of 1 unit.
	//  3. Distribute the remaining (12 - cols) units proportionally by the
	//     capped length.  The last column absorbs any integer-division remainder.
	const maxCap = 50
	capLen := make([]int, cols)
	for i, l := range maxLen {
		if l > maxCap {
			capLen[i] = maxCap
		} else {
			capLen[i] = l
		}
	}
	totalCap := 0
	for _, l := range capLen {
		totalCap += l
	}
	budget := 12 - cols // extra units beyond the 1-unit floor
	if budget < 0 {
		budget = 0
	}
	colSizes := make([]int, cols)
	assigned := 0
	for i := 0; i < cols-1; i++ {
		extra := 0
		if totalCap > 0 && budget > 0 {
			extra = (capLen[i] * budget) / totalCap
		}
		colSizes[i] = 1 + extra
		assigned += colSizes[i]
	}
	last := 12 - assigned
	if last < 1 {
		last = 1
	}
	colSizes[cols-1] = last

	const fontSize = 9.0
	for _, tr := range trows {
		// Row height = height needed by the cell that wraps the most lines.
		maxLines := 1
		for i := 0; i < cols; i++ {
			txt := ""
			if i < len(tr.cells) {
				txt = tr.cells[i]
			}
			if l := estimateLines(txt, colSizes[i], fontSize); l > maxLines {
				maxLines = l
			}
		}
		ht := rowHeight(maxLines, 7)

		style := fontstyle.Normal
		if tr.isHeader {
			style = fontstyle.Bold
		}
		mcols := make([]core.Col, 0, cols)
		for i := 0; i < cols; i++ {
			txt := ""
			if i < len(tr.cells) {
				txt = tr.cells[i]
			}
			mcols = append(mcols, marototext.NewCol(colSizes[i], txt, props.Text{
				Size:  fontSize,
				Style: style,
				Align: align.Left,
			}))
		}
		r.rows = append(r.rows, marotorow.New(ht).Add(mcols...))
	}
}
