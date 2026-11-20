package dto

import "time"

// UploadResult is a purpose-bound upload token, usable until expiresAt (one hour, D18).
type UploadResult struct {
	Token      string    `json:"token"`
	ExpiresAt  time.Time `json:"expiresAt"`
	ServerTime int64     `json:"serverTime"`
}
