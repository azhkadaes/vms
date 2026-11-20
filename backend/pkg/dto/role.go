package dto

// RoleDefinition is one fixed role as the role administration screen shows it (ops 30, 31).
type RoleDefinition struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Permissions           []string `json:"permissions"`
	AssignablePermissions []string `json:"assignablePermissions"`
	Editable              bool     `json:"editable"`
	Version               int32    `json:"version"`
}

// RoleUpdate replaces a role's permissions when version is still current (op 31).
type RoleUpdate struct {
	Permissions []string `json:"permissions"`
	Version     int32    `json:"version"`
}
