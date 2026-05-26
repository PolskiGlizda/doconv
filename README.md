# doconv

A fast, self-contained CLI for converting documents between common formats.
**No external binaries required** — everything runs in-process with pure Go libraries.
When Google Chrome or Chromium is detected on the system, HTML → PDF output is automatically upgraded to use it for higher-fidelity rendering.

---

## Supported conversions

| From | To |
|------|----|
| Markdown (`.md`) | HTML, PDF, EPUB, DOCX |
| HTML (`.html`) | Markdown, PDF, EPUB |
| DOCX (`.docx`) | HTML, Markdown |

PDF → anything is not supported — PDF text extraction requires a C library (pdfium) and would break the no-binary-dependency guarantee.

---

## Installation

### From source

```bash
git clone https://github.com/PolskiGlizda/doconv.git
cd doconv
go install -ldflags="-X main.version=$(git describe --tags --always)" ./cmd/doconv
```

Requires **Go 1.21+**.

### Pre-built binaries

Download from the [Releases](https://github.com/PolskiGlizda/doconv/releases) page.

---

## Usage

```
doconv convert <input> <output> [flags]
doconv formats
doconv version
```

### Basic examples

```bash
# Markdown → PDF
doconv convert README.md output.pdf

# Markdown → HTML
doconv convert README.md output.html

# Markdown → EPUB
doconv convert README.md book.epub

# Markdown → DOCX
doconv convert README.md document.docx

# HTML → Markdown
doconv convert page.html page.md

# HTML → PDF  (uses Chrome if available, otherwise pure-Go fallback)
doconv convert page.html page.pdf

# DOCX → Markdown
doconv convert report.docx report.md

# DOCX → HTML
doconv convert report.docx report.html
```

### Override format detection

By default the format is detected from the file extension. Use `--from` / `--to` to override:

```bash
doconv convert input --from md --to html > output.html
doconv convert styles.html output.pdf --to pdf
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--from FORMAT` | | Source format override (`md`, `html`, `docx`, `pdf`, `epub`) |
| `--to FORMAT` | | Target format override |
| `--verbose` | `-v` | Print conversion details and file size to stderr |
| `--help` | `-h` | Help for any command |

### List all supported routes

```bash
doconv formats
```

---

## HTML → PDF: Chrome vs pure-Go fallback

doconv probes for a Chrome/Chromium binary at startup (`google-chrome`, `google-chrome-stable`, `chromium`, `chromium-browser`, `microsoft-edge`). 

- **Chrome found** — uses headless Chrome via CDP for pixel-perfect, CSS-aware PDF rendering.
- **Chrome not found** — converts HTML → Markdown → PDF using the pure-Go pipeline. Output is clean and portable but won't reproduce complex CSS layouts.

No configuration needed; the upgrade is automatic.

---

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Conversion error or file not found |
| `2` | Unsupported format or bad `--from`/`--to` value |

---

## Libraries used

| Library | Purpose | License |
|---------|---------|---------|
| [yuin/goldmark](https://github.com/yuin/goldmark) | Markdown → HTML | MIT |
| [JohannesKaufmann/html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown) | HTML → Markdown | MIT |
| [johnfercher/maroto](https://github.com/johnfercher/maroto) | PDF generation | MIT |
| [fumiama/go-docx](https://github.com/fumiama/go-docx) | DOCX read/write | AGPL-3.0 |
| [bmaupin/go-epub](https://github.com/bmaupin/go-epub) | EPUB generation | MIT |
| [gabriel-vasile/mimetype](https://github.com/gabriel-vasile/mimetype) | MIME detection | MIT |
| [chromedp/chromedp](https://github.com/chromedp/chromedp) | Headless Chrome (optional) | MIT |
| [schollz/progressbar](https://github.com/schollz/progressbar) | Spinner / progress | MIT |
| [spf13/cobra](https://github.com/spf13/cobra) | CLI framework | Apache-2.0 |

---

## License

doconv is distributed under the [GNU Affero General Public License v3.0](LICENSE).

The AGPL requirement is driven by the `fumiama/go-docx` dependency. In practice, for open-source use (source publicly available on GitHub) there are no additional obligations beyond keeping the source open. If you need a more permissive license, remove the DOCX converters (`internal/converter/docx*.go`, `internal/converter/md2docx.go`) and all remaining code is MIT-compatible.
