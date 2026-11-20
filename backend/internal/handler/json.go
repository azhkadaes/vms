package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"

	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// DecodeJSON rejects non-JSON content (415), unknown fields and trailing values (400).
// Use dto.Optional for members that must distinguish omitted, null and a value.
func DecodeJSON(c fiber.Ctx, target any) error {
	media, _, err := mime.ParseMediaType(c.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return failure.New(415, "unsupported_media_type")
	}
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return failure.New(400, "invalid_request")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return failure.New(400, "invalid_request")
	}
	return nil
}

// NoQuery rejects any query string on operations that take none.
func NoQuery(c fiber.Ctx) error {
	if len(c.Request().URI().QueryString()) > 0 {
		return failure.New(400, "invalid_query_parameter")
	}
	return nil
}
