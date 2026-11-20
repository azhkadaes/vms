package config

import (
	"bytes"
	"encoding/base64"
	"errors"
	"path/filepath"
	"strconv"
)

// Uploads configures private upload storage (D15) and the dedicated token key (D99).
// A nil Uploads keeps uploads disabled.
type Uploads struct {
	// Dir is the absolute private directory the local storage adapter writes to.
	Dir string
	// TokenKey derives upload tokens; KeyVersion is stored with each upload so a replay
	// never re-derives its token with another key.
	TokenKey   []byte
	KeyVersion int16
}

// uploadsFrom enables uploads only when both the directory and the token key are configured.
// The token key must differ from every other configured key.
func uploadsFrom(get func(string) string, others ...[]byte) (*Uploads, error) {
	dir := get("UPLOAD_DIR")
	if dir == "" && get("UPLOAD_TOKEN_KEY") == "" && get("UPLOAD_TOKEN_KEY_FILE") == "" && get("UPLOAD_TOKEN_KEY_VERSION") == "" {
		return nil, nil
	}
	if dir == "" || !filepath.IsAbs(dir) {
		return nil, errors.New("UPLOAD_DIR must be an absolute private directory when uploads are configured")
	}
	encoded, err := secret(get, "UPLOAD_TOKEN_KEY")
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) < 32 {
		return nil, errors.New("UPLOAD_TOKEN_KEY must be base64 encoding at least 32 random bytes")
	}
	for _, other := range others {
		if bytes.Equal(key, other) {
			return nil, errors.New("UPLOAD_TOKEN_KEY must be independent of the rate-limit and authentication keys")
		}
	}
	version := int64(1)
	if raw := get("UPLOAD_TOKEN_KEY_VERSION"); raw != "" {
		version, err = strconv.ParseInt(raw, 10, 16)
		if err != nil || version < 1 {
			return nil, errors.New("UPLOAD_TOKEN_KEY_VERSION must be 1-32767")
		}
	}
	return &Uploads{Dir: filepath.Clean(dir), TokenKey: key, KeyVersion: int16(version)}, nil
}
