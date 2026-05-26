package apperr

import "fmt"

// UnsupportedFormatError is returned when a source→target format pair has no
// registered converter. The CLI exits with code 2.
type UnsupportedFormatError struct {
	From string
	To   string
}

func (e *UnsupportedFormatError) Error() string {
	if e.To == "" {
		return fmt.Sprintf("unsupported format: %q", e.From)
	}
	return fmt.Sprintf("no converter for %s → %s", e.From, e.To)
}

// ConversionError wraps an underlying error from a converter with the file
// context. The CLI exits with code 1.
type ConversionError struct {
	SourceFile   string
	TargetFormat string
	Cause        error
}

func (e *ConversionError) Error() string {
	return fmt.Sprintf("failed to convert %q to %s: %v", e.SourceFile, e.TargetFormat, e.Cause)
}

func (e *ConversionError) Unwrap() error { return e.Cause }
