package dto

import "time"

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type PasswordChange struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}
type RoleSelect struct {
	Role string `json:"role"`
}
type Refresh struct {
	RefreshRequestID string `json:"refreshRequestId"`
}
type TokenResult struct {
	AccessToken     string    `json:"accessToken"`
	TokenType       string    `json:"tokenType"`
	AccessExpiresAt time.Time `json:"accessExpiresAt"`
	Session         Session   `json:"session"`
}
