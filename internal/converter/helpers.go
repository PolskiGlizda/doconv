package converter

import "bytes"

// newReaderAt wraps a byte slice in a *bytes.Reader, which satisfies io.ReaderAt.
// Used by converters that call docx.Parse(readerAt, size).
func newReaderAt(data []byte) *bytes.Reader {
	return bytes.NewReader(data)
}
