package converter

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestMDToHTML(t *testing.T) {
	t.Parallel()
	conv, err := Get("md", "html")
	if err != nil {
		t.Fatalf("get converter: %v", err)
	}
	src := strings.NewReader("# Hello\n\nWorld paragraph.\n")
	var dst bytes.Buffer
	if err := conv.Convert(context.Background(), src, &dst); err != nil {
		t.Fatalf("convert: %v", err)
	}
	out := dst.String()
	if !strings.Contains(out, "<h1>") {
		t.Errorf("expected <h1> tag, got:\n%s", out)
	}
	if !strings.Contains(out, "World paragraph") {
		t.Errorf("expected paragraph text, got:\n%s", out)
	}
}

func TestHTMLToMD(t *testing.T) {
	t.Parallel()
	conv, err := Get("html", "md")
	if err != nil {
		t.Fatalf("get converter: %v", err)
	}
	src := strings.NewReader("<h1>Hello</h1><p>World paragraph.</p>")
	var dst bytes.Buffer
	if err := conv.Convert(context.Background(), src, &dst); err != nil {
		t.Fatalf("convert: %v", err)
	}
	out := dst.String()
	if !strings.Contains(out, "Hello") {
		t.Errorf("expected heading text, got:\n%s", out)
	}
	if !strings.Contains(out, "World paragraph") {
		t.Errorf("expected paragraph text, got:\n%s", out)
	}
}

func TestMDToPDF(t *testing.T) {
	t.Parallel()
	conv, err := Get("md", "pdf")
	if err != nil {
		t.Fatalf("get converter: %v", err)
	}
	src := strings.NewReader("# Title\n\nParagraph text.\n\n- item one\n- item two\n")
	var dst bytes.Buffer
	if err := conv.Convert(context.Background(), src, &dst); err != nil {
		t.Fatalf("convert: %v", err)
	}
	// PDF magic bytes
	if !bytes.HasPrefix(dst.Bytes(), []byte("%PDF")) {
		t.Errorf("output does not start with PDF magic bytes, got: %q", dst.Bytes()[:8])
	}
}

func TestMDToEPUB(t *testing.T) {
	t.Parallel()
	conv, err := Get("md", "epub")
	if err != nil {
		t.Fatalf("get converter: %v", err)
	}
	src := strings.NewReader("# Book Title\n\nChapter content here.\n")
	var dst bytes.Buffer
	if err := conv.Convert(context.Background(), src, &dst); err != nil {
		t.Fatalf("convert: %v", err)
	}
	// EPUB is a ZIP; magic bytes are PK
	if !bytes.HasPrefix(dst.Bytes(), []byte("PK")) {
		t.Errorf("output does not start with ZIP/EPUB magic bytes")
	}
}

func TestMDToDOCX(t *testing.T) {
	t.Parallel()
	conv, err := Get("md", "docx")
	if err != nil {
		t.Fatalf("get converter: %v", err)
	}
	src := strings.NewReader("# Heading\n\nParagraph with **bold** and _italic_ text.\n")
	var dst bytes.Buffer
	if err := conv.Convert(context.Background(), src, &dst); err != nil {
		t.Fatalf("convert: %v", err)
	}
	// DOCX is also a ZIP
	if !bytes.HasPrefix(dst.Bytes(), []byte("PK")) {
		t.Errorf("output does not start with ZIP/DOCX magic bytes")
	}
}

func TestHTMLToEPUB(t *testing.T) {
	t.Parallel()
	conv, err := Get("html", "epub")
	if err != nil {
		t.Fatalf("get converter: %v", err)
	}
	src := strings.NewReader("<h1>Title</h1><p>Content.</p>")
	var dst bytes.Buffer
	if err := conv.Convert(context.Background(), src, &dst); err != nil {
		t.Fatalf("convert: %v", err)
	}
	if !bytes.HasPrefix(dst.Bytes(), []byte("PK")) {
		t.Errorf("output does not start with ZIP/EPUB magic bytes")
	}
}

func TestHTMLToPDF_Fallback(t *testing.T) {
	t.Parallel()
	// This test exercises the pure-Go fallback (HTML→MD→PDF).
	// Chrome won't be present in CI; the fallback must not error.
	conv, err := Get("html", "pdf")
	if err != nil {
		t.Fatalf("get converter: %v", err)
	}
	src := strings.NewReader("<h1>Page Title</h1><p>Body text here.</p>")
	var dst bytes.Buffer
	if err := conv.Convert(context.Background(), src, &dst); err != nil {
		t.Fatalf("convert: %v", err)
	}
	if !bytes.HasPrefix(dst.Bytes(), []byte("%PDF")) {
		t.Errorf("output does not start with PDF magic bytes")
	}
}

func TestUnsupportedFormat(t *testing.T) {
	t.Parallel()
	_, err := Get("xyz", "pdf")
	if err == nil {
		t.Fatal("expected error for unsupported format, got nil")
	}
}

func TestRoutes(t *testing.T) {
	routes := Routes()
	if len(routes) == 0 {
		t.Fatal("expected at least one registered route")
	}
	// Spot-check a few expected routes
	for _, want := range []string{"md→html", "html→md", "md→pdf", "md→epub"} {
		found := false
		for _, r := range routes {
			if r == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("route %q not registered; routes: %v", want, routes)
		}
	}
}
