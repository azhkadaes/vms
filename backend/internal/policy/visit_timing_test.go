package policy_test

import (
	"testing"
	"time"

	"github.com/azhkadaes/vms/backend/internal/policy"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

func TestVisitTiming(t *testing.T) {
	start := time.Date(2030, 2, 1, 10, 0, 0, 0, validate.WITA)
	const duration = 60
	for _, tt := range []struct {
		name, status, display     string
		offset                    time.Duration
		checkIn, checkOut, cancel bool
	}{
		{"pending before start", "pending", "pending", -time.Millisecond, false, false, true},
		{"before opening", "approved", "approved", -30*time.Minute - time.Millisecond, false, false, true},
		{"at opening", "approved", "approved", -30 * time.Minute, true, false, true},
		{"at start", "approved", "approved", 0, true, false, false},
		{"at end", "approved", "approved", time.Hour, true, false, false},
		{"past end", "approved", "missed", time.Hour + time.Millisecond, false, false, false},
		{"overnight checkout", "checked_in", "checked_in", 24 * time.Hour, false, true, false},
		{"rejected before start", "rejected", "rejected", -time.Hour, false, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			now := start.Add(tt.offset)
			want := dto.BookingTiming{
				ServerTime: now.UnixMilli(), OpensAt: start.Add(-30 * time.Minute).UnixMilli(),
				EndsAt: start.Add(time.Hour).UnixMilli(), DisplayStatus: tt.display,
				CanCheckIn: tt.checkIn, CanCheckOut: tt.checkOut, CanCancel: tt.cancel,
			}
			if got := policy.VisitTiming(tt.status, start, duration, now); got != want {
				t.Fatalf("timing = %+v, want %+v", got, want)
			}
		})
	}
}
