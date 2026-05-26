// Package converter provides the central registry and interface for all
// document format converters.
package converter

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/PolskiGlizda/doconv/internal/apperr"
)

// Converter converts a document from one format to another.
type Converter interface {
	Convert(ctx context.Context, src io.Reader, dst io.Writer) error
}

// ConverterFunc is a function adapter for Converter.
type ConverterFunc func(ctx context.Context, src io.Reader, dst io.Writer) error

func (f ConverterFunc) Convert(ctx context.Context, src io.Reader, dst io.Writer) error {
	return f(ctx, src, dst)
}

// key encodes a source→target route.
type key struct{ from, to string }

var registry = map[key]Converter{}

// Register adds a converter for the given from→to pair.
// Panics if a converter is already registered for that pair (programming error).
func Register(from, to string, c Converter) {
	k := key{from, to}
	if _, dup := registry[k]; dup {
		panic(fmt.Sprintf("converter: duplicate registration for %s→%s", from, to))
	}
	registry[k] = c
}

// Get returns the converter for from→to, or an UnsupportedFormatError.
func Get(from, to string) (Converter, error) {
	if c, ok := registry[key{from, to}]; ok {
		return c, nil
	}
	return nil, &apperr.UnsupportedFormatError{From: from, To: to}
}

// Routes returns a sorted list of "from→to" strings for all registered pairs.
func Routes() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k.from+"→"+k.to)
	}
	sort.Strings(out)
	return out
}
