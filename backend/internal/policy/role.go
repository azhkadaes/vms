package policy

// Role names exactly as seeded in vms.role; a session's active role is one of these.
const (
	RolePA           = "PA"
	RoleProtocol     = "Protocol Officer"
	RoleReceptionist = "Receptionist"
	RoleSuperAdmin   = "Super Admin"
)

// Permission keys exactly as seeded in vms.permission (the A-17 catalogue).
const (
	PermBookings     = "bookings"
	PermCreate       = "create"
	PermApprove      = "approve"
	PermReject       = "reject"
	PermReschedule   = "reschedule"
	PermCancel       = "cancel"
	PermCheckin      = "checkin"
	PermCheckout     = "checkout"
	PermReports      = "reports"
	PermReportExport = "reportExport"
	PermPeople       = "people"
	PermRoles        = "roles"
	PermIntegrations = "integrations"
	PermAudit        = "audit"
)
