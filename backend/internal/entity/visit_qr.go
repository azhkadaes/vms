package entity

import "time"

type VisitQR struct {
	ID        int32
	VisitID   int32
	TokenHash string
	CreatedAt time.Time
	ExpiresAt time.Time
}
