package dto

// Employee is one employee in the administration screens (ops 26-29). Passwords never appear;
// profile fields are omitted when empty.
type Employee struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Email              string   `json:"email"`
	Roles              []string `json:"roles"`
	JobTitle           string   `json:"jobTitle,omitempty"`
	Deputy             string   `json:"deputy,omitempty"`
	Directorate        string   `json:"directorate,omitempty"`
	Phone              string   `json:"phone,omitempty"`
	PhoneNumbers       []string `json:"phoneNumbers"`
	NIP                string   `json:"nip,omitempty"`
	Active             bool     `json:"active"`
	AssignedOfficerIDs []string `json:"assignedOfficerIds"`
	Version            int32    `json:"version"`
}

// EmployeePage is op 26: Total counts the filtered employees, TotalAll every employee.
type EmployeePage struct {
	Items             []Employee `json:"items"`
	Total             int        `json:"total"`
	TotalAll          int        `json:"totalAll"`
	Page              int        `json:"page"`
	PageSize          int        `json:"pageSize"`
	PasswordMinLength int        `json:"passwordMinLength"`
}

// EmployeeCreate is op 28. Absent, null or "" profile fields stay empty. Active and
// requirePasswordChange must be sent explicitly.
type EmployeeCreate struct {
	Name                  string   `json:"name"`
	Email                 string   `json:"email"`
	Roles                 []string `json:"roles"`
	JobTitle              string   `json:"jobTitle"`
	Deputy                string   `json:"deputy"`
	Directorate           string   `json:"directorate"`
	Phone                 string   `json:"phone"`
	PhoneNumbers          []string `json:"phoneNumbers"`
	NIP                   string   `json:"nip"`
	Active                *bool    `json:"active"`
	AssignedOfficerIDs    []string `json:"assignedOfficerIds"`
	InitialPassword       string   `json:"initialPassword"`
	RequirePasswordChange *bool    `json:"requirePasswordChange"`
}

// EmployeeUpdate is op 29: an omitted field keeps its value, "" or null clears a profile field,
// and an omitted or null roles or assignedOfficerIds list keeps the current one. Version is required.
type EmployeeUpdate struct {
	Name               Optional[string]   `json:"name"`
	Email              Optional[string]   `json:"email"`
	Roles              []string           `json:"roles"`
	JobTitle           Optional[string]   `json:"jobTitle"`
	Deputy             Optional[string]   `json:"deputy"`
	Directorate        Optional[string]   `json:"directorate"`
	Phone              Optional[string]   `json:"phone"`
	PhoneNumbers       Optional[[]string] `json:"phoneNumbers"`
	NIP                Optional[string]   `json:"nip"`
	Active             Optional[bool]     `json:"active"`
	AssignedOfficerIDs []string           `json:"assignedOfficerIds"`
	Version            int32              `json:"version"`
}
