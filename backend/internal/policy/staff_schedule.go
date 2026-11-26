package policy

import (
	"time"

	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

// StaffScheduleStart validates a new start against the WITA booking window and database time.
func StaffScheduleStart(date, clock string, now time.Time) (time.Time, bool) {
	start, err := validate.StaffStart(date, clock)
	return start, err == nil && start.After(now)
}
