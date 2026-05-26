// Package detect resolves a file path to a canonical format name ("md", "html",
// "pdf", "docx", "epub") using file-extension lookup first and a MIME-magic
// fallback for extensionless or ambiguous files.
package detect

import (
	"path/filepath"
	"strings"

	"github.com/PolskiGlizda/doconv/internal/apperr"
	"github.com/gabriel-vasile/mimetype"
)

// extMap maps lower-case file extensions (without leading dot) to the
// canonical format name used throughout the converter registry.
var extMap = map[string]string{
	"md":       "md",
	"markdown": "md",
	"html":     "html",
	"htm":      "html",
	"pdf":      "pdf",
	"docx":     "docx",
	"doc":      "docx",
	"epub":     "epub",
	"txt":      "txt",
}

// mimeMap maps MIME type strings to canonical format names.
var mimeMap = map[string]string{
	"text/markdown": "md",
	"text/x-markdown": "md",
	"text/html":     "html",
	"application/pdf": "pdf",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "docx",
	"application/epub+zip": "epub",
	"text/plain":           "txt",
}

// FromPath returns the canonical format name for filePath.
// It first tries the file extension, then falls back to MIME magic byte sniffing.
// Returns UnsupportedFormatError when the format cannot be determined.
func FromPath(filePath string) (string, error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filePath)), ".")
	if f, ok := extMap[ext]; ok {
		return f, nil
	}

	// MIME fallback (reads only the first 512 bytes)
	mt, err := mimetype.DetectFile(filePath)
	if err != nil {
		return "", &apperr.UnsupportedFormatError{From: ext}
	}
	// Walk MIME hierarchy (e.g. "text/html; charset=utf-8" → "text/html")
	for m := mt; m != nil; m = m.Parent() {
		if f, ok := mimeMap[m.String()]; ok {
			return f, nil
		}
		// strip parameters
		base := strings.SplitN(m.String(), ";", 2)[0]
		base = strings.TrimSpace(base)
		if f, ok := mimeMap[base]; ok {
			return f, nil
		}
	}

	return "", &apperr.UnsupportedFormatError{From: mt.String()}
}

// Canonical returns the canonical name for a user-supplied format string
// (e.g. "markdown" → "md", "htm" → "html").
func Canonical(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if f, ok := extMap[s]; ok {
		return f, nil
	}
	return "", &apperr.UnsupportedFormatError{From: s}
}
