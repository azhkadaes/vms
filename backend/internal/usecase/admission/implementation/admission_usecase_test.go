package admissionusecaseimplementation_test

import (
	"context"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/azhkadaes/vms/backend/internal/entity"
	admission "github.com/azhkadaes/vms/backend/internal/usecase/admission/implementation"
)

type repository struct{ keys []string }

func (r *repository) PublicRequest(ctx context.Context, key string) (entity.RequestAdmission, error) {
	r.keys = append(r.keys, key)
	return entity.RequestAdmission{Allowed: true}, nil
}
func TestLimiterIdentityIsCanonicalAndKeyed(t *testing.T) {
	r := &repository{}
	u := admission.NewAdmissionUsecase(r, []byte(strings.Repeat("k", 32)), noop.NewTracerProvider().Tracer("test"))
	for _, ip := range []string{"203.0.113.5", "::ffff:203.0.113.5", "203.0.113.6"} {
		if _, err := u.PublicRequest(context.Background(), ip); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.keys[0]) != 64 || r.keys[0] != r.keys[1] || r.keys[0] == r.keys[2] || strings.Contains(r.keys[0], "203.0.113.5") {
		t.Fatal("address aliasing or raw-IP persistence")
	}
	if _, err := u.PublicRequest(context.Background(), "spoofed"); err == nil || len(r.keys) != 3 {
		t.Fatal("invalid source reached persistence")
	}
}
