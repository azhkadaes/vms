package config

import (
	"bytes"
	"encoding/base64"
	"errors"
)

func admissionQRKeyFrom(get func(string) string, others [][]byte) ([]byte, error) {
	const name = "VMS_ADMISSION_QR_KEY"
	if get(name) == "" && get(name+"_FILE") == "" {
		return nil, nil
	}
	value, err := secret(get, name)
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(key) < 32 {
		return nil, errors.New("VMS_ADMISSION_QR_KEY must be base64 encoding at least 32 random bytes")
	}
	for _, other := range others {
		if bytes.Equal(key, other) {
			return nil, errors.New("VMS_ADMISSION_QR_KEY must be independent of other security keys")
		}
	}
	return key, nil
}
