package policy

import (
	"strings"
	"unicode"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
	"github.com/azhkadaes/vms/backend/pkg/helper/security"
)

const (
	AttendanceCheckIn  = "check_in"
	AttendanceCheckOut = "check_out"
)

func ReceptionCredential(raw string) (entity.ReceptionCredential, error) {
	value := strings.TrimSpace(raw)
	if value == "" || len(value) > 1024 || strings.ContainsFunc(value, unicode.IsControl) {
		return entity.ReceptionCredential{}, failure.Validation().Add("credential", "Masukkan kode atau pindai QR yang valid.").Err()
	}
	if len(value) == 8 && !strings.ContainsFunc(value, func(r rune) bool { return r < '0' || r > '9' }) {
		return entity.ReceptionCredential{Code: value}, nil
	}
	return entity.ReceptionCredential{TokenHash: security.Hash(value)}, nil
}

func CanResolveReception(session dto.Session, status string) bool {
	switch status {
	case "approved":
		return CanAttend(session, PermCheckin)
	case "checked_in":
		return CanAttend(session, PermCheckout)
	}
	return false
}
