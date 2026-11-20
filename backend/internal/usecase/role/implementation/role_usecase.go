package roleusecaseimplementation

import (
	"context"
	"fmt"
	"slices"

	"go.opentelemetry.io/otel/trace"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	rolerepository "github.com/azhkadaes/vms/backend/internal/repository/role"
	roleserializer "github.com/azhkadaes/vms/backend/internal/serializer/role"
	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	roleusecase "github.com/azhkadaes/vms/backend/internal/usecase/role"
	"github.com/azhkadaes/vms/backend/pkg/dto"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

type usecase struct {
	repo       rolerepository.RoleRepository
	serializer roleserializer.RoleSerializer
	validator  authusecase.SessionValidator
	tracer     trace.Tracer
}

var _ roleusecase.RoleUsecase = (*usecase)(nil)

// NewRoleUsecase composes role administration with the shared session validator for write rechecks.
func NewRoleUsecase(repo rolerepository.RoleRepository, serializer roleserializer.RoleSerializer, validator authusecase.SessionValidator, tracer trace.Tracer) roleusecase.RoleUsecase {
	return &usecase{repo: repo, serializer: serializer, validator: validator, tracer: tracer}
}

// GetRoles answers 403 before any read when the session lacks the roles permission.
func (u *usecase) GetRoles(ctx context.Context, session dto.Session) ([]dto.RoleDefinition, error) {
	ctx, span := u.tracer.Start(ctx, "role.list")
	defer span.End()
	if !policy.Has(session, policy.PermRoles) {
		return nil, failure.New(403, "forbidden")
	}
	roles, err := u.repo.GetRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	return u.serializer.ToRoleDefinitions(roles), nil
}

// UpdateRole runs authority, role lock, version and A-17 checks in that order inside one
// transaction, so a refused save writes nothing. A save that changes nothing writes nothing
// either: no version bump, and live sessions keep their role revision.
func (u *usecase) UpdateRole(ctx context.Context, proof entity.SessionProof, id int32, req dto.RoleUpdate) (dto.RoleDefinition, error) {
	ctx, span := u.tracer.Start(ctx, "role.update")
	defer span.End()
	var out entity.Role
	err := u.repo.Transaction(ctx, func(tx rolerepository.RoleTransaction) error {
		snapshot, err := tx.LockAuthority(proof)
		if err != nil {
			return err
		}
		session, err := u.validator.Validate(snapshot, proof)
		if err != nil {
			return err
		}
		if !policy.Has(session, policy.PermRoles) {
			return failure.New(403, "forbidden")
		}
		role, err := tx.LockRole(id)
		if err != nil {
			return err
		}
		if err := validUpdate(req); err != nil {
			return err
		}
		if role.Version != req.Version {
			return failure.New(409, "version_conflict")
		}
		if !policy.RoleEditable(role.Name, role.PolicyID > 0) {
			return failure.New(409, "invalid_state")
		}
		if !policy.KeepsFixedPermissions(role.Name, role.Permissions, req.Permissions) {
			return failure.Validation().Add("permissions", "Izin tetap peran ini tidak dapat diubah.").Err()
		}
		out = role
		if sameSet(role.Permissions, req.Permissions) {
			return nil
		}
		at := snapshot.ServerTime
		if out, err = tx.SavePermissions(role, req.Permissions, snapshot.EmployeeID, at); err != nil {
			return err
		}
		if err := tx.BumpRoleRevision(role.ID, at); err != nil {
			return err
		}
		return tx.AuditPermissionChange(snapshot.EmployeeID, role, out, at)
	})
	if err != nil {
		return dto.RoleDefinition{}, fmt.Errorf("update role: %w", err)
	}
	return u.serializer.ToRoleDefinition(out), nil
}

// validUpdate checks the request on its own: every permission a known key, once, and a version.
func validUpdate(req dto.RoleUpdate) error {
	fields := failure.Validation()
	if req.Permissions == nil {
		fields.Add("permissions", "Daftar izin wajib diisi.")
	}
	for i, key := range req.Permissions {
		if !policy.KnownPermission(key) {
			fields.Add("permissions", "Pilih izin dari daftar yang tersedia.")
		} else if slices.Contains(req.Permissions[:i], key) {
			fields.Add("permissions", "Setiap izin hanya boleh dipilih sekali.")
		}
	}
	if req.Version <= 0 {
		fields.Add("version", "Versi data tidak valid. Muat ulang lalu coba lagi.")
	}
	return fields.Err()
}

// sameSet compares two duplicate-free permission lists ignoring order.
func sameSet(a, b []string) bool {
	return len(a) == len(b) && !slices.ContainsFunc(b, func(key string) bool { return !slices.Contains(a, key) })
}
