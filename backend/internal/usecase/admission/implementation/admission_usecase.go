package admissionusecaseimplementation

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/netip"

	"go.opentelemetry.io/otel/trace"

	"github.com/azhkadaes/vms/backend/internal/entity"
	admissionrepository "github.com/azhkadaes/vms/backend/internal/repository/admission"
	admissionusecase "github.com/azhkadaes/vms/backend/internal/usecase/admission"
)

type usecase struct {
	repo   admissionrepository.AdmissionRepository
	key    []byte
	tracer trace.Tracer
}

var _ admissionusecase.AdmissionUsecase = (*usecase)(nil)

// NewAdmissionUsecase privately copies the dedicated key used to keep source IPs out of stored counters.
func NewAdmissionUsecase(repo admissionrepository.AdmissionRepository, key []byte, tracer trace.Tracer) admissionusecase.AdmissionUsecase {
	return &usecase{repo: repo, key: append([]byte(nil), key...), tracer: tracer}
}

// PublicRequest maps IPv4 aliases to one keyed digest and rejects invalid proxy input.
func (u *usecase) PublicRequest(ctx context.Context, ip string) (entity.RequestAdmission, error) {
	ctx, span := u.tracer.Start(ctx, "admission.public")
	defer span.End()
	address, err := netip.ParseAddr(ip)
	if err != nil {
		return entity.RequestAdmission{}, errors.New("trusted request address unavailable")
	}
	mac := hmac.New(sha256.New, u.key)
	_, _ = mac.Write([]byte("vms:public:requests:v1\x00" + address.Unmap().String()))
	return u.repo.PublicRequest(ctx, hex.EncodeToString(mac.Sum(nil)))
}
