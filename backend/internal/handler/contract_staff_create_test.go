package handler_test

import (
	"context"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

type contractStaffCreator struct{}

func (contractStaffCreator) CreateStaffBooking(_ context.Context, session dto.Session, proof entity.SessionProof, _ string, input dto.StaffBookingCreate) (dto.StaffBooking, bool, error) {
	if !policy.Has(session, policy.PermCreate) || proof.EmployeeID <= 0 {
		return dto.StaffBooking{}, false, failure.New(403, "forbidden")
	}
	if input.Time == "10:05" {
		return dto.StaffBooking{}, false, failure.New(409, "schedule_conflict")
	}
	booking := dto.StaffBooking{
		ID: "7", Date: input.Date, Time: input.Time, Duration: 60, Room: "", Status: "approved", Version: 1,
		VisitKind: "restricted", Reference: "TRB-7", RestrictedView: "full",
		Timing:  dto.BookingTiming{ServerTime: 1794794400000, OpensAt: 1794879000000, EndsAt: 1794884400000, DisplayStatus: "approved", CanCancel: true},
		Officer: &dto.BookingOfficer{Name: "Pejabat Contoh"}, OfficerID: "3",
		CreatedBy: &dto.StaffBookingActor{Name: "Petugas Contoh", Role: policy.RoleProtocol},
	}
	return booking, input.IdempotencyKey != "12345678-1234-4234-8234-123456789abb", nil
}
