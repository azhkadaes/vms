package handler

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

const (
	// multipartOverhead covers boundaries, part headers and the two short text parts.
	multipartOverhead = 16 << 10
	maxTextPartBytes  = 256
)

// UploadForm is one bounded upload request: the file bytes plus its reference and purpose.
// Content is not verified here; the usecase checks signatures against the purpose.
type UploadForm struct {
	File      []byte
	Filename  string
	Reference string
	Purpose   string
}

// Multipart reads multipart/form-data holding exactly the parts file, reference and purpose.
// A declared body over the cap is rejected with 413 before any of it is read, and the read
// itself is bounded, so a missing or false Content-Length cannot exceed the cap either.
func Multipart(c fiber.Ctx, maxFileBytes int64) (UploadForm, error) {
	media, params, err := mime.ParseMediaType(c.Get("Content-Type"))
	if err != nil || media != "multipart/form-data" || params["boundary"] == "" {
		return UploadForm{}, failure.New(415, "unsupported_media_type")
	}
	limit := maxFileBytes + multipartOverhead
	if int64(c.Request().Header.ContentLength()) > limit {
		return UploadForm{}, failure.New(413, "payload_too_large")
	}
	var source io.Reader
	if c.Request().IsBodyStream() {
		source = c.Request().BodyStream()
	} else {
		source = bytes.NewReader(c.Body())
	}
	body := &io.LimitedReader{R: source, N: limit + 1}
	reader := multipart.NewReader(body, params["boundary"])
	readFailure := func() error {
		if body.N <= 0 {
			return failure.New(413, "payload_too_large")
		}
		return failure.New(400, "invalid_request")
	}

	var out UploadForm
	seen := map[string]bool{}
	invalid := failure.Validation()
	for {
		part, err := reader.NextRawPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return UploadForm{}, readFailure()
		}
		name := part.FormName()
		if seen[name] || (name != "file" && name != "reference" && name != "purpose") {
			return UploadForm{}, failure.New(400, "invalid_request")
		}
		seen[name] = true
		limitBytes := int64(maxTextPartBytes)
		if name == "file" {
			limitBytes = maxFileBytes
		}
		data, err := io.ReadAll(io.LimitReader(part, limitBytes+1))
		if err != nil {
			return UploadForm{}, readFailure()
		}
		if int64(len(data)) > limitBytes {
			if name == "file" {
				return UploadForm{}, failure.New(413, "payload_too_large")
			}
			invalid.Add(name, "Nilai tidak valid.")
			continue
		}
		switch name {
		case "file":
			out.File, out.Filename = data, part.FileName()
		case "reference":
			out.Reference = string(data)
		case "purpose":
			out.Purpose = string(data)
		}
		if name != "file" && !utf8.Valid(data) {
			invalid.Add(name, "Nilai tidak valid.")
		}
	}
	if !seen["file"] {
		invalid.Add("file", "Pilih berkas yang akan diunggah.")
	} else if len(out.File) == 0 {
		invalid.Add("file", "Berkas tidak boleh kosong.")
	}
	if out.Reference == "" && !invalid.Has("reference") {
		invalid.Add("reference", "Referensi unggahan wajib diisi.")
	}
	if out.Purpose == "" && !invalid.Has("purpose") {
		invalid.Add("purpose", "Jenis berkas wajib dipilih.")
	}
	if err := invalid.Err(); err != nil {
		return UploadForm{}, err
	}
	return out, nil
}
