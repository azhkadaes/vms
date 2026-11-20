package employeeserializerimplementation

import (
	"strconv"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	employeeserializer "github.com/azhkadaes/vms/backend/internal/serializer/employee"
	"github.com/azhkadaes/vms/backend/pkg/dto"
)

type serializer struct{}

var _ employeeserializer.EmployeeSerializer = (*serializer)(nil)

// NewEmployeeSerializer projects employees; it has no configuration.
func NewEmployeeSerializer() employeeserializer.EmployeeSerializer { return &serializer{} }

// ToEmployee emits string IDs, roles[] (never a singular role) and copies of every list. Empty
// profile fields are omitted; credentials and the forced-change flag are not part of the view.
func (s *serializer) ToEmployee(in entity.Employee) dto.Employee {
	officers := make([]string, 0, len(in.OfficerIDs))
	for _, id := range in.OfficerIDs {
		officers = append(officers, strconv.FormatInt(int64(id), 10))
	}
	return dto.Employee{ID: strconv.FormatInt(int64(in.ID), 10), Name: in.Name, Email: in.Email, Roles: append([]string{}, in.Roles...),
		JobTitle: in.JobTitle, Deputy: in.Deputy, Directorate: in.Directorate, Phone: in.Phone, NIP: in.NIP,
		Active: in.Active, AssignedOfficerIDs: officers, Version: in.Version}
}

// ToEmployeePage echoes the effective paging; a page past the end has empty items, never null.
func (s *serializer) ToEmployeePage(in entity.EmployeePage, filter entity.EmployeeFilter) dto.EmployeePage {
	items := make([]dto.Employee, 0, len(in.Items))
	for _, employee := range in.Items {
		items = append(items, s.ToEmployee(employee))
	}
	return dto.EmployeePage{Items: items, Total: in.Total, TotalAll: in.TotalAll, Page: filter.Page, PageSize: filter.PageSize, PasswordMinLength: policy.PasswordMinLength}
}
