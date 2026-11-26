package dto

import "time"

type Admission struct {
	ImagePNG   string    `json:"imagePng"`
	Payload    string    `json:"payload"`
	ExpiresAt  time.Time `json:"expiresAt"`
	ServerTime int64     `json:"serverTime"`
}
