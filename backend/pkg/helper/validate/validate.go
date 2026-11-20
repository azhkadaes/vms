package validate

import (
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/google/uuid"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func ID(value string) (int32, error) {
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil || n <= 0 || strconv.FormatInt(n, 10) != value {
		return 0, failure.New(400, "invalid_request")
	}
	return int32(n), nil
}
func Reference(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id != uuid.Nil && id.String() == value
}
func Text(value string, min, max int) bool {
	return utf8.ValidString(value) && value == strings.TrimSpace(value) && utf8.RuneCountInString(value) >= min && utf8.RuneCountInString(value) <= max
}

// Email normalizes an address the way vms.pegawai stores it: trimmed and lower-case, printable
// ASCII only, at most 254 bytes, and a bare address (no display name). ok is false otherwise.
func Email(raw string) (string, bool) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if len(email) == 0 || len(email) > 254 {
		return "", false
	}
	for _, c := range email {
		if c > 127 || c < 33 {
			return "", false
		}
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", false
	}
	return email, true
}
func Digits(value string, min, max int) bool {
	if len(value) < min || len(value) > max {
		return false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// WITA is fixed UTC+8; no machine-local timezone participates in business dates.
var WITA = time.FixedZone("Asia/Makassar", 8*60*60)

func Start(date, clock string) (time.Time, error) {
	value, err := time.ParseInLocation("2006-01-02 15:04", date+" "+clock, WITA)
	if err != nil || value.Format("2006-01-02 15:04") != date+" "+clock {
		return time.Time{}, failure.Validation().Add("date", "Pilih tanggal yang valid.").Err()
	}
	if clock < "08:00" || clock > "16:00" {
		return time.Time{}, failure.Validation().Add("time", "Pilih waktu mulai antara 08.00–16.00 WITA.").Err()
	}
	return value.UTC(), nil
}
