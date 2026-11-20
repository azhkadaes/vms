package uploadusecaseimplementation

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/azhkadaes/vms/backend/internal/policy"
)

const (
	// maxPhotoPixels bounds the decoded size of a photo, so a small file cannot expand into a huge bitmap.
	maxPhotoPixels = 25_000_000
	// maxNameRunes matches form_documents.nama and the frontend's file-name rule.
	maxNameRunes = 255
	// pdfTrailerBytes is how close to the end a PDF's final %%EOF must be.
	pdfTrailerBytes = 1024
)

var (
	pdfHeader     = regexp.MustCompile(`^%PDF-[12]\.[0-9]`)
	xrefTable     = regexp.MustCompile(`^xref\s+[0-9]+[ \t]+[1-9][0-9]*\s+[0-9]{10} [0-9]{5} [fn]\s`)
	xrefTrailer   = regexp.MustCompile(`\btrailer\s*<<`)
	xrefStream    = regexp.MustCompile(`^[0-9]+\s+[0-9]+\s+obj\s*<<`)
	xrefType      = regexp.MustCompile(`/Type\s*/XRef\b`)
	xrefData      = regexp.MustCompile(`>>\s*stream\r?\n`)
	jpegSignature = []byte{0xff, 0xd8, 0xff}
	pngSignature  = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
)

// inspect identifies content by its signature and proves it decodes as that type; the client's
// declared type and file name play no part. It returns the verified MIME type when purpose
// accepts it, otherwise a field message for file.
func inspect(content []byte, purpose string) (mime, problem string) {
	switch {
	case bytes.HasPrefix(content, []byte("%PDF-")):
		if wellFormedPDF(content) {
			mime = "application/pdf"
		}
	case bytes.HasPrefix(content, jpegSignature):
		mime, problem = decodePhoto(content, "image/jpeg", jpeg.DecodeConfig, jpeg.Decode)
	case bytes.HasPrefix(content, pngSignature):
		mime, problem = decodePhoto(content, "image/png", png.DecodeConfig, png.Decode)
	}
	if policy.AcceptsUpload(purpose, mime) {
		return mime, ""
	}
	if purpose == policy.PurposeDocument {
		return "", "Gunakan berkas PDF yang valid."
	}
	if problem != "" {
		return "", problem
	}
	return "", "Gunakan foto JPEG atau PNG yang valid."
}

// wellFormedPDF checks the header, the final %%EOF and that startxref points at a
// cross-reference table or stream, which a renamed or truncated file does not have.
func wellFormedPDF(content []byte) bool {
	if !pdfHeader.Match(content[:min(len(content), 16)]) {
		return false
	}
	end := bytes.LastIndex(content, []byte("%%EOF"))
	if end < 0 || len(content)-end > pdfTrailerBytes {
		return false
	}
	start := bytes.LastIndex(content[:end], []byte("startxref"))
	if start < 0 {
		return false
	}
	offset, err := strconv.Atoi(strings.TrimSpace(string(content[start+len("startxref") : end])))
	if err != nil || offset <= 0 || offset >= start {
		return false
	}
	section := bytes.TrimLeft(content[offset:start], " \t\r\n\f\x00")
	if xrefTable.Match(section) {
		return xrefTrailer.Match(section)
	}
	streamHeader := xrefData.FindIndex(section)
	return streamHeader != nil && xrefStream.Match(section) && xrefType.Match(section[:streamHeader[0]]) &&
		bytes.Contains(section[streamHeader[1]:], []byte("endstream"))
}

// decodePhoto decodes the whole image after its header shows a bounded size.
func decodePhoto(content []byte, mime string, header func(io.Reader) (image.Config, error), decode func(io.Reader) (image.Image, error)) (string, string) {
	config, err := header(bytes.NewReader(content))
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return "", ""
	}
	if int64(config.Width)*int64(config.Height) > maxPhotoPixels {
		return "", "Resolusi foto maksimal 25 megapiksel."
	}
	if _, err := decode(bytes.NewReader(content)); err != nil {
		return "", ""
	}
	return mime, ""
}

// defaultNames name a file whose client name is empty after cleaning.
var defaultNames = map[string]string{"application/pdf": "nota-dinas.pdf", "image/jpeg": "foto.jpg", "image/png": "foto.png"}

// fileName keeps the client's base name without directories, control characters, invalid
// UTF-8 or surrounding spaces; ok is false when the cleaned name is over 255 characters.
func fileName(raw, mime string) (name string, ok bool) {
	name = strings.ToValidUTF8(raw, "")
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	name = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name))
	if name == "" {
		return defaultNames[mime], true
	}
	return name, utf8.RuneCountInString(name) <= maxNameRunes
}
