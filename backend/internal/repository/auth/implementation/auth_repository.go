package authrepositoryimplementation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repo struct{ db *gorm.DB }

var _ authrepository.AuthRepository = (*repo)(nil)

func NewAuthRepository(db *gorm.DB) authrepository.AuthRepository { return &repo{db: db} }

// GetSessionSnapshot reads all authority from one snapshot without renewing activity.
func (r *repo) GetSessionSnapshot(ctx context.Context, sessionID string) (entity.SessionSnapshot, error) {
	parsed, err := uuid.Parse(sessionID)
	if err != nil || parsed == uuid.Nil || parsed.String() != sessionID {
		return entity.SessionSnapshot{}, gorm.ErrRecordNotFound
	}
	out := entity.SessionSnapshot{AvailableRoles: []string{}, Permissions: []string{}, AssignedOfficerIDs: []int32{}}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Raw(`SELECT s.employee_id, e.nama AS name, e.status_aktif AS account_active,
			e.must_change_password, COALESCE(r.nama, '') AS active_role,
			s.credential_generation, s.role_revision, s.created_at, s.last_activity_at, s.idle_expires_at,
			s.absolute_expires_at, s.revoked_at, transaction_timestamp() AS server_time
			FROM support.staff_sessions s JOIN vms.pegawai e ON e.id = s.employee_id
			LEFT JOIN vms.role r ON r.id = s.active_role_id WHERE s.id = ?`, sessionID).Scan(&out)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Table("vms.role_pegawai AS a").Select("r.nama").Joins("JOIN vms.role r ON r.id = a.id_role").Where("a.id_karyawan = ?", out.EmployeeID).Order("r.id").Pluck("r.nama", &out.AvailableRoles).Error; err != nil {
			return err
		}
		if out.ActiveRole == "" {
			return nil
		}
		if err := tx.Raw(`SELECT DISTINCT p.permission_key FROM support.staff_sessions s
			JOIN vms.role_pegawai a ON a.id_karyawan = s.employee_id AND a.id_role = s.active_role_id
			JOIN vms.role_policy rp ON rp.id_role = a.id_role AND rp.status_aktif
			JOIN vms.policy policy ON policy.id = rp.id_policy AND policy.status_aktif
			JOIN vms.policy_permission pp ON pp.id_policy = policy.id
			JOIN vms.permission p ON p.id = pp.id_permission
			WHERE s.id = ? ORDER BY p.permission_key`, sessionID).Scan(&out.Permissions).Error; err != nil {
			return err
		}
		// Directorate metadata never grants scope. Inactive officials/assignments do not grant scope.
		return tx.Table("vms.pegawai_pejabat AS a").Joins("JOIN vms.pejabat p ON p.id = a.id_pejabat AND p.status_aktif").Where("a.id_pegawai = ? AND a.status_aktif", out.EmployeeID).Order("a.id_pejabat").Pluck("a.id_pejabat", &out.AssignedOfficerIDs).Error
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.SessionSnapshot{}, gorm.ErrRecordNotFound
	}
	if err != nil {
		return entity.SessionSnapshot{}, fmt.Errorf("read session authority: %w", err)
	}
	return out, nil
}
