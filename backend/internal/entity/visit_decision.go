package entity

type VisitSchedule struct {
	OfficerID    int32
	RoomID       *int32
	Date         string
	Time         string
	Duration     int
	Instructions *string
	Coordination *string
	ServiceID    *string
}

type VisitChange struct {
	Status    string
	AuditType string
	Message   string
	Reason    string
	Internal  *string
	Schedule  *VisitSchedule
	Identity  *RestrictedIdentity
	RevokeQR  bool
}

type RestrictedIdentity struct {
	Name         *string
	Organization *string
}
