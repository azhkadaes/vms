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

// snapshotSelect and snapshotFrom bracket the clock that a snapshot is validated against.
const snapshotSelect = `SELECT s.employee_id, e.nama AS name, e.status_aktif AS account_active,
	e.must_change_password, COALESCE(r.nama, '') AS active_role,
	s.credential_generation, s.role_revision, s.created_at, s.last_activity_at, s.idle_expires_at,
	s.absolute_expires_at, s.revoked_at, `
const snapshotFrom = ` AS server_time
	FROM support.staff_sessions s JOIN vms.pegawai e ON e.id = s.employee_id
	LEFT JOIN vms.role r ON r.id = s.active_role_id WHERE s.id = ?`

// readSnapshotSQL times a read-only snapshot at the instant its snapshot was taken.
const readSnapshotSQL = snapshotSelect + "transaction_timestamp()" + snapshotFrom

// lockedSnapshotSQL times a write transaction's recheck after the employee lock, which may have waited.
const lockedSnapshotSQL = snapshotSelect + "clock_timestamp()" + snapshotFrom

// canonicalSession keeps malformed session identifiers away from the database.
func canonicalSession(id string) bool {
	parsed, err := uuid.Parse(id)
	return err == nil && parsed != uuid.Nil && parsed.String() == id
}

// GetSessionSnapshot reads all authority from one snapshot without renewing activity.
func (r *repo) GetSessionSnapshot(ctx context.Context, sessionID string) (entity.SessionSnapshot, error) {
	if !canonicalSession(sessionID) {
		return entity.SessionSnapshot{}, gorm.ErrRecordNotFound
	}
	var out entity.SessionSnapshot
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var found bool
		var err error
		out, found, err = readSnapshot(tx, readSnapshotSQL, sessionID)
		if err == nil && !found {
			return gorm.ErrRecordNotFound
		}
		return err
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.SessionSnapshot{}, gorm.ErrRecordNotFound
	}
	if err != nil {
		return entity.SessionSnapshot{}, fmt.Errorf("read session authority: %w", err)
	}
	return out, nil
}

// readSnapshot reads one session's authority on tx; found is false when the session row is absent.
func readSnapshot(tx *gorm.DB, query, sessionID string) (entity.SessionSnapshot, bool, error) {
	out := entity.SessionSnapshot{AvailableRoles: []string{}, Permissions: []string{}, AssignedOfficerIDs: []int32{}}
	result := tx.Raw(query, sessionID).Scan(&out)
	if result.Error != nil {
		return entity.SessionSnapshot{}, false, result.Error
	}
	if result.RowsAffected != 1 {
		return entity.SessionSnapshot{}, false, nil
	}
	if err := tx.Table("vms.role_pegawai AS a").Select("r.nama").Joins("JOIN vms.role r ON r.id = a.id_role").Where("a.id_karyawan = ?", out.EmployeeID).Order("r.id").Pluck("r.nama", &out.AvailableRoles).Error; err != nil {
		return entity.SessionSnapshot{}, false, err
	}
	if out.ActiveRole == "" {
		return out, true, nil
	}
	if err := tx.Raw(`SELECT DISTINCT p.permission_key FROM support.staff_sessions s
		JOIN vms.role_pegawai a ON a.id_karyawan = s.employee_id AND a.id_role = s.active_role_id
		JOIN vms.role_policy rp ON rp.id_role = a.id_role AND rp.status_aktif
		JOIN vms.policy policy ON policy.id = rp.id_policy AND policy.status_aktif
		JOIN vms.policy_permission pp ON pp.id_policy = policy.id
		JOIN vms.permission p ON p.id = pp.id_permission
		WHERE s.id = ? ORDER BY p.permission_key`, sessionID).Scan(&out.Permissions).Error; err != nil {
		return entity.SessionSnapshot{}, false, err
	}
	// Directorate metadata never grants scope. Inactive officials/assignments do not grant scope.
	if err := tx.Table("vms.pegawai_pejabat AS a").Joins("JOIN vms.pejabat p ON p.id = a.id_pejabat AND p.status_aktif").Where("a.id_pegawai = ? AND a.status_aktif", out.EmployeeID).Order("a.id_pejabat").Pluck("a.id_pejabat", &out.AssignedOfficerIDs).Error; err != nil {
		return entity.SessionSnapshot{}, false, err
	}
	return out, true, nil
}
