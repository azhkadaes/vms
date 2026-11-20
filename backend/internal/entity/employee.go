package entity

import "time"

// Employee is an employee as administration reads and writes it. Credentials are never part of it.
type Employee struct {
	ID                 int32
	Name               string
	Email              string
	JobTitle           string
	Deputy             string
	Directorate        string
	Phone              string
	NIP                string
	Active             bool
	MustChangePassword bool
	Version            int32
	// Roles are role names in vms.role ID order.
	Roles []string
	// OfficerIDs are the officials of the employee's active PA assignment (at most one), whatever
	// the officials' own status.
	OfficerIDs []int32
}

// EmployeeFilter selects one page of the employee list. Query matches name, email or NIP.
type EmployeeFilter struct {
	Query    string
	Page     int
	PageSize int
}

// EmployeePage is one page of employees with the filtered (Total) and unfiltered (TotalAll) counts.
type EmployeePage struct {
	Items    []Employee
	Total    int
	TotalAll int
}

// EmployeeOfficer is an official as a PA assignment check reads it.
type EmployeeOfficer struct {
	ID     int32
	VIP    bool `gorm:"column:vip"`
	Active bool
}

// EmployeeEvent is one audited employee change: the staff actor, the employee target and the
// state before (nil on creation) and after.
type EmployeeEvent struct {
	Actor  int32
	Action string
	Before *Employee
	After  Employee
	At     time.Time
}
