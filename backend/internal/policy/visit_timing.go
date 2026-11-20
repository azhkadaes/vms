package policy

import (
	"time"

	"github.com/azhkadaes/vms/backend/pkg/dto"
)

func VisitTiming(status string, start time.Time, duration int, now time.Time) dto.BookingTiming {
	opens := start.Add(-30 * time.Minute)
	ends := start.Add(time.Duration(duration) * time.Minute)
	return dto.BookingTiming{
		ServerTime: now.UnixMilli(), OpensAt: opens.UnixMilli(), EndsAt: ends.UnixMilli(),
		DisplayStatus: DisplayStatus(status, start, duration, now),
		CanCheckIn:    status == "approved" && !now.Before(opens) && !now.After(ends),
		CanCheckOut:   status == "checked_in",
		CanCancel:     (status == "pending" || status == "approved") && now.Before(start),
	}
}
