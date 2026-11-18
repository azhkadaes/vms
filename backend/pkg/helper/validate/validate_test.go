package validate_test

import (
	"errors"
	"testing"
	"time"

	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/validate"
)

func TestStartReportsDateAndTimeAsFormFieldErrors(t *testing.T) {
	for _, tc := range []struct{ date, clock, key string }{
		{"2026-02-30", "09:00", "date"},
		{"2026-11-18", "16:01", "time"},
		{"2026-11-18", "07:59", "time"},
	} {
		_, err := validate.Start(tc.date, tc.clock)
		var got *failure.Error
		if !errors.As(err, &got) || got.Status != 422 || got.Code != "validation_failed" || len(got.Fields[tc.key]) != 1 || len(got.Fields) != 1 {
			t.Fatalf("%s %s: %v", tc.date, tc.clock, err)
		}
	}
	start, err := validate.Start("2026-11-18", "16:00")
	if err != nil || !start.Equal(time.Date(2026, 11, 18, 8, 0, 0, 0, time.UTC)) || start.Location() != time.UTC {
		t.Fatalf("got %v %v", start, err)
	}
}
