package rolerepositoryimplementation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authrepositoryimplementation "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	rolerepository "github.com/azhkadaes/vms/backend/internal/repository/role"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

type repo struct{ db *gorm.DB }

var _ rolerepository.RoleRepository = (*repo)(nil)

func NewRoleRepository(db *gorm.DB) rolerepository.RoleRepository { return &repo{db: db} }

// roleSelect reads a role with its active policy: both the role_policy link and the policy must
// be active, as in the session authority read. policy_id is 0 when the role has none.
const roleSelect = `SELECT r.id, r.nama AS name, r.version, COALESCE(p.id, 0) AS policy_id
	FROM vms.role r
	LEFT JOIN vms.role_policy rp ON rp.id_role = r.id AND rp.status_aktif
	LEFT JOIN vms.policy p ON p.id = rp.id_policy AND p.status_aktif`

type roleRow struct {
	ID       int32
	Name     string
	Version  int32
	PolicyID int32
}

type grantRow struct {
	PolicyID int32
	Key      string
}

// withPermissions reads the grants of every row's policy on db, in catalogue order.
func withPermissions(db *gorm.DB, rows []roleRow) ([]entity.Role, error) {
	policies := []int32{}
	for _, row := range rows {
		if row.PolicyID > 0 {
			policies = append(policies, row.PolicyID)
		}
	}
	grants := []grantRow{}
	if len(policies) > 0 {
		if err := db.Raw(`SELECT pp.id_policy AS policy_id, k.permission_key AS key FROM vms.policy_permission pp
			JOIN vms.permission k ON k.id = pp.id_permission WHERE pp.id_policy IN ? ORDER BY pp.id_policy, k.id`, policies).Scan(&grants).Error; err != nil {
			return nil, err
		}
	}
	out := make([]entity.Role, 0, len(rows))
	for _, row := range rows {
		role := entity.Role{ID: row.ID, Name: row.Name, Version: row.Version, PolicyID: row.PolicyID, Permissions: []string{}}
		for _, grant := range grants {
			if row.PolicyID > 0 && grant.PolicyID == row.PolicyID {
				role.Permissions = append(role.Permissions, grant.Key)
			}
		}
		out = append(out, role)
	}
	return out, nil
}

// GetRoles reads roles and grants from one REPEATABLE READ, READ ONLY snapshot.
func (r *repo) GetRoles(ctx context.Context) ([]entity.Role, error) {
	var out []entity.Role
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows := []roleRow{}
		if err := tx.Raw(roleSelect + " ORDER BY r.id").Scan(&rows).Error; err != nil {
			return err
		}
		var err error
		out, err = withPermissions(tx, rows)
		return err
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("read roles: %w", err)
	}
	return out, nil
}

type roleTx struct {
	authrepository.AuthorityLock
	db *gorm.DB
}

var _ rolerepository.RoleTransaction = (*roleTx)(nil)

// Transaction serializes staff access changes before fn takes its first row lock.
func (r *repo) Transaction(ctx context.Context, fn func(rolerepository.RoleTransaction) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := authrepositoryimplementation.LockAccessWriters(tx); err != nil {
			return err
		}
		return fn(&roleTx{AuthorityLock: authrepositoryimplementation.NewAuthorityLock(tx), db: tx})
	})
}

func (t *roleTx) LockRole(id int32) (entity.Role, error) {
	rows := []roleRow{}
	if err := t.db.Raw(roleSelect+" WHERE r.id = ? FOR UPDATE OF r", id).Scan(&rows).Error; err != nil {
		return entity.Role{}, fmt.Errorf("lock role: %w", err)
	}
	if len(rows) != 1 {
		return entity.Role{}, fmt.Errorf("lock role %d: %w", id, rolerepository.ErrNotFound)
	}
	roles, err := withPermissions(t.db, rows)
	if err != nil {
		return entity.Role{}, fmt.Errorf("read locked role permissions: %w", err)
	}
	return roles[0], nil
}

// SavePermissions deletes the policy's grants outside permissions and inserts the missing ones,
// then moves the version on only from role.Version. updated_by is the actor's employee ID as text.
func (t *roleTx) SavePermissions(role entity.Role, permissions []string, actor int32, at time.Time) (entity.Role, error) {
	if role.PolicyID <= 0 {
		return entity.Role{}, fmt.Errorf("save role %d permissions: no active policy", role.ID)
	}
	if len(permissions) == 0 {
		if err := t.db.Exec("DELETE FROM vms.policy_permission WHERE id_policy = ?", role.PolicyID).Error; err != nil {
			return entity.Role{}, fmt.Errorf("withdraw role permissions: %w", err)
		}
	} else {
		if err := t.db.Exec(`DELETE FROM vms.policy_permission pp USING vms.permission k
			WHERE k.id = pp.id_permission AND pp.id_policy = ? AND k.permission_key NOT IN ?`, role.PolicyID, permissions).Error; err != nil {
			return entity.Role{}, fmt.Errorf("withdraw role permissions: %w", err)
		}
		granted := t.db.Exec(`INSERT INTO vms.policy_permission(id_permission, id_policy)
			SELECT k.id, ? FROM vms.permission k WHERE k.permission_key IN ? ORDER BY k.id
			ON CONFLICT (id_policy, id_permission) DO NOTHING`, role.PolicyID, permissions)
		if granted.Error != nil {
			return entity.Role{}, fmt.Errorf("grant role permissions: %w", granted.Error)
		}
	}
	bumped := t.db.Exec(`UPDATE vms.role SET version = version + 1, updated_at = ?, updated_by = ?
		WHERE id = ? AND version = ? AND version < 2147483647`, at, strconv.FormatInt(int64(actor), 10), role.ID, role.Version)
	if bumped.Error != nil {
		return entity.Role{}, fmt.Errorf("bump role version: %w", bumped.Error)
	}
	if bumped.RowsAffected != 1 {
		return entity.Role{}, failure.New(409, "version_conflict")
	}
	rows := []roleRow{}
	if err := t.db.Raw(roleSelect+" WHERE r.id = ?", role.ID).Scan(&rows).Error; err != nil {
		return entity.Role{}, fmt.Errorf("read saved role: %w", err)
	}
	if len(rows) != 1 {
		return entity.Role{}, fmt.Errorf("read saved role %d: expected one row, found %d", role.ID, len(rows))
	}
	saved, err := withPermissions(t.db, rows)
	if err != nil {
		return entity.Role{}, fmt.Errorf("read saved role permissions: %w", err)
	}
	if len(saved[0].Permissions) != len(permissions) {
		return entity.Role{}, fmt.Errorf("save role %d permissions: %d of %d keys stored", role.ID, len(saved[0].Permissions), len(permissions))
	}
	return saved[0], nil
}

// liveOnRole selects the sessions whose active role is ? and that can still make a request at ?.
const liveOnRole = `active_role_id = ? AND revoked_at IS NULL AND idle_expires_at > ? AND absolute_expires_at > ?`

// BumpRoleRevision locks every employee assigned to the role before finding live sessions.
// A session can activate the role while we wait for its employee lock.
func (t *roleTx) BumpRoleRevision(roleID int32, at time.Time) error {
	var locked []int32
	if err := t.db.Raw(`SELECT e.id FROM vms.pegawai e JOIN vms.role_pegawai rp ON rp.id_karyawan = e.id
		WHERE rp.id_role = ? ORDER BY e.id FOR UPDATE OF e`, roleID).Scan(&locked).Error; err != nil {
		return fmt.Errorf("lock role holders: %w", err)
	}
	if err := t.db.Exec(`UPDATE support.staff_sessions SET role_revision = role_revision + 1 WHERE `+liveOnRole, roleID, at, at).Error; err != nil {
		return fmt.Errorf("bump role revision: %w", err)
	}
	return nil
}

type permissionValues struct {
	Permissions []string `json:"permissions"`
	Version     int32    `json:"version"`
}

func (t *roleTx) AuditPermissionChange(actor int32, before, after entity.Role, at time.Time) error {
	values := func(role entity.Role) (string, error) {
		raw, err := json.Marshal(permissionValues{Permissions: append([]string{}, role.Permissions...), Version: role.Version})
		return string(raw), err
	}
	old, err := values(before)
	if err != nil {
		return fmt.Errorf("audit role change: %w", err)
	}
	updated, err := values(after)
	if err != nil {
		return fmt.Errorf("audit role change: %w", err)
	}
	if err := t.db.Exec(`INSERT INTO vms.audit_log(actor_type, id_pegawai, entity_type, entity_id, jenis_kegiatan, tanggal, deskripsi, old_values, new_values)
		VALUES ('staff', ?, 'role', ?, 'role.permissions_changed', ?, ?, ?::json, ?::json)`,
		actor, before.ID, at, "Role permissions changed: "+before.Name, old, updated).Error; err != nil {
		return fmt.Errorf("audit role change: %w", err)
	}
	return nil
}
