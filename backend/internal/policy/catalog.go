package policy

import "github.com/azhkadaes/vms/backend/pkg/dto"

// StaffCatalogScope is the part of the staff catalog (op 08) a session may read, beyond the
// active officers and rooms every allowed session gets (A-15).
type StaffCatalogScope struct {
	// AssignedInactiveOfficers adds inactive officers an employee is still assigned to, so the
	// employee editor can show an assignment it can no longer offer.
	AssignedInactiveOfficers bool
}

// StaffCatalog grants op 08 to a ready session holding bookings or people; ok is false otherwise.
// Officers carry vip, so booking forms narrow their own options instead of the server.
func StaffCatalog(s dto.Session) (scope StaffCatalogScope, ok bool) {
	people := Has(s, PermPeople)
	if !people && !Has(s, PermBookings) {
		return StaffCatalogScope{}, false
	}
	return StaffCatalogScope{AssignedInactiveOfficers: people}, true
}
