package dto

type PublicLookup struct {
	Code  string `json:"code"`
	Email string `json:"email"`
}

type PublicCancel struct {
	Code    string `json:"code"`
	Email   string `json:"email"`
	Version int    `json:"version"`
}
