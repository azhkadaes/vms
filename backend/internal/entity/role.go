package entity

// Role is one of the four fixed roles with the permission keys its active policy grants.
type Role struct {
	ID      int32
	Name    string
	Version int32
	// PolicyID is the role's active policy, which holds its permissions; 0 when it has none.
	PolicyID int32
	// Permissions are in catalogue (vms.permission) order.
	Permissions []string
}
