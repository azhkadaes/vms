package entity

import "time"

// ReceptionCredential carries exactly one proof. QR values are hashed before reaching the repository.
type ReceptionCredential struct {
	Code      string
	TokenHash string
}

type ReceptionMatch struct {
	VisitID     int32
	QRExpiresAt *time.Time
}
