package entity

// RequestAdmission carries a shared-budget decision and a retry delay in seconds.
type RequestAdmission struct {
	Allowed    bool
	RetryAfter int
}
