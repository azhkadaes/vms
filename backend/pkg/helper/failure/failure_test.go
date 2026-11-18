package failure_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

func TestValidationCollectsSeveralMessagesPerFlatKey(t *testing.T) {
	err := failure.Validation().
		Add("visitors", "Nama pengunjung wajib diisi.").
		Add("time", "Pilih waktu mulai antara 08.00–16.00 WITA.").
		Add("visitors", "Maksimal 10 pengunjung.").
		Add("visitors", "Nama pengunjung wajib diisi.").
		Err()
	var got *failure.Error
	if !errors.As(err, &got) {
		t.Fatalf("not a failure: %v", err)
	}
	if got.Status != 422 || got.Code != "validation_failed" {
		t.Fatalf("got %d %s", got.Status, got.Code)
	}
	want := map[string][]string{
		"visitors": {"Nama pengunjung wajib diisi.", "Maksimal 10 pengunjung."},
		"time":     {"Pilih waktu mulai antara 08.00–16.00 WITA."},
	}
	if !reflect.DeepEqual(got.Fields, want) {
		t.Fatalf("fields %v", got.Fields)
	}
}

func TestValidationWithoutFieldsIsNoError(t *testing.T) {
	if err := failure.Validation().Err(); err != nil {
		t.Fatalf("empty validation returned %v", err)
	}
}

func TestValidationErrorIsDetachedFromLaterAdds(t *testing.T) {
	builder := failure.Validation().Add("email", "Masukkan alamat email yang valid.")
	var first *failure.Error
	errors.As(builder.Err(), &first)
	builder.Add("email", "Email sudah terdaftar.").Add("phone", "Nomor telepon tidak valid.")
	if len(first.Fields) != 1 || len(first.Fields["email"]) != 1 {
		t.Fatalf("earlier error changed: %v", first.Fields)
	}
}

func TestValidationNeverEmitsEmptyMessages(t *testing.T) {
	var got *failure.Error
	errors.As(failure.Validation().Add("note", "").Err(), &got)
	if len(got.Fields["note"]) != 1 || got.Fields["note"][0] == "" {
		t.Fatalf("empty message kept: %v", got.Fields)
	}
}

// The catalogue is the conventions §5 table plus the two transport codes the
// error handler already emits; each must have its own user-facing message.
func TestEveryCatalogueCodeHasItsOwnMessage(t *testing.T) {
	codes := []string{
		"invalid_request", "invalid_query_parameter",
		"unauthenticated", "invalid_credentials",
		"forbidden", "csrf_expired",
		"not_found",
		"method_not_allowed",
		"version_conflict", "invalid_state", "schedule_conflict", "reschedule_window_closed", "idempotency_conflict",
		"notice_outdated", "credential_expired", "revision_conflict", "role_changed",
		"payload_too_large", "unsupported_media_type",
		"validation_failed", "export_too_large",
		"rate_limited",
		"internal_error", "service_unavailable",
	}
	seen := map[string]string{}
	for _, code := range codes {
		message, ok := failure.Message(code)
		if !ok || message == "" {
			t.Fatalf("no message for %s", code)
		}
		if other, dup := seen[message]; dup {
			t.Fatalf("%s and %s share a message", code, other)
		}
		seen[message] = code
	}
	if _, ok := failure.Message("made_up_code"); ok {
		t.Fatal("unknown code has a message")
	}
}

func TestMessagesUsedVerbatimByTheUI(t *testing.T) {
	for code, want := range map[string]string{
		"validation_failed": "Periksa isian yang ditandai.",
		"unauthenticated":   "Sesi berakhir. Silakan masuk kembali.",
		"version_conflict":  "Data telah berubah. Muat ulang sebelum melanjutkan.",
		"schedule_conflict": "Ruangan atau pejabat memiliki jadwal yang bertabrakan.",
	} {
		if got, _ := failure.Message(code); got != want {
			t.Errorf("%s: %q", code, got)
		}
	}
}

func TestNotFoundSurvivesRepositoryWrapping(t *testing.T) {
	moduleSentinel := fmt.Errorf("booking: %w", failure.ErrNotFound)
	err := fmt.Errorf("lock visit 7: %w", moduleSentinel)
	if !errors.Is(err, failure.ErrNotFound) || !errors.Is(err, moduleSentinel) {
		t.Fatal("wrapped not-found lost its identity")
	}
}

func TestCauseIsKeptForLogsButNotInTheCode(t *testing.T) {
	cause := errors.New("dial tcp 10.0.0.5:5432: connection refused")
	err := failure.New(503, "service_unavailable").WithCause(cause)
	if err.Error() != "service_unavailable" {
		t.Fatalf("message leaked cause: %q", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatal("cause not reachable for logging")
	}
}
