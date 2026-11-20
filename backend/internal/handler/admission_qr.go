package handler

import (
	"bytes"
	_ "embed"
	"image/png"

	"github.com/piglig/go-qr/v2"
)

//go:embed ikn-qr-logo.png
var admissionLogoPNG []byte

var admissionLogo, admissionLogoErr = png.Decode(bytes.NewReader(admissionLogoPNG))

func renderAdmissionQR(payload string) ([]byte, error) {
	if admissionLogoErr != nil {
		return nil, admissionLogoErr
	}
	code, err := qr.Encode(payload, qr.WithECC(qr.ECCHigh))
	if err != nil {
		return nil, err
	}
	return code.PNG(qr.WithLogo(admissionLogo, 0.18))
}
