package employeerepositoryimplementation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	"github.com/azhkadaes/vms/backend/internal/policy"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	authimplementation "github.com/azhkadaes/vms/backend/internal/repository/auth/implementation"
	employeerepository "github.com/azhkadaes/vms/backend/internal/repository/employee"
)

type transaction struct {
	authrepository.AuthorityLock
	db *gorm.DB
}

var _ employeerepository.EmployeeTransaction = (*transaction)(nil)

func (r *repository) Transaction(ctx context.Context, fn func(employeerepository.EmployeeTransaction) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := authimplementation.LockAccessWriters(tx); err != nil {
			return err
		}
		return fn(&transaction{AuthorityLock: authimplementation.NewAuthorityLock(tx), db: tx})
	})
}

func (t *transaction) LockEmployee(id int32) (entity.Employee, error) {
	return getEmployee(t.db, id, true)
}

func (t *transaction) EmailInUse(email string, except int32) (bool, error) {
	var exists bool
	err := t.db.Raw("SELECT EXISTS (SELECT 1 FROM vms.pegawai WHERE email = ? AND id <> ?)", email, except).Scan(&exists).Error
	return exists, err
}

func (t *transaction) Officers(ids []int32) ([]entity.EmployeeOfficer, error) {
	var out []entity.EmployeeOfficer
	err := t.db.Raw(`SELECT p.id, COALESCE(j.pejabat_vip, FALSE) AS vip, p.status_aktif AS active
		FROM vms.pejabat p LEFT JOIN vms.jabatan j ON j.id = p.id_jabatan WHERE p.id IN ?`, ids).Scan(&out).Error
	return out, err
}

func (t *transaction) OtherActiveSuperAdmins(except int32) (int, error) {
	var count int
	err := t.db.Raw(`SELECT count(*) FROM vms.pegawai e JOIN vms.role_pegawai a ON a.id_karyawan = e.id
		JOIN vms.role r ON r.id = a.id_role WHERE e.status_aktif AND e.id <> ? AND r.nama = ?`,
		except, policy.RoleSuperAdmin).Scan(&count).Error
	return count, err
}

func employeeWriteError(err error) error {
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) && postgres.Code == "23505" && postgres.ConstraintName == "pegawai_email_key" {
		return employeerepository.ErrEmailTaken
	}
	return err
}

func (t *transaction) CreateEmployee(employee entity.Employee, hash string) (entity.Employee, error) {
	var inserted struct {
		ID      int32
		Version int32
	}
	result := t.db.Raw(`INSERT INTO vms.pegawai(nama,email,jabatan,kedeputian,direktorat,nomor_hp,nip,status_aktif,must_change_password)
		VALUES (?,?,NULLIF(?,''),NULLIF(?,''),NULLIF(?,''),NULLIF(?,''),NULLIF(?,''),?,?) RETURNING id, version`,
		employee.Name, employee.Email, employee.JobTitle, employee.Deputy, employee.Directorate,
		employee.Phone, employee.NIP, employee.Active, employee.MustChangePassword).Scan(&inserted)
	if result.Error != nil {
		return entity.Employee{}, fmt.Errorf("insert employee: %w", employeeWriteError(result.Error))
	}
	if result.RowsAffected != 1 || inserted.ID <= 0 {
		return entity.Employee{}, errors.New("insert employee: no row returned")
	}
	employee.ID, employee.Version = inserted.ID, inserted.Version
	if err := t.db.Exec("INSERT INTO support.employee_credentials(employee_id,encoded_hash) VALUES (?,?)", employee.ID, hash).Error; err != nil {
		return entity.Employee{}, fmt.Errorf("insert employee credential: %w", err)
	}
	if err := t.saveAccess(entity.Employee{}, employee); err != nil {
		return entity.Employee{}, err
	}
	return employee, nil
}

func (t *transaction) SaveEmployee(before, after entity.Employee) (entity.Employee, error) {
	result := t.db.Exec(`UPDATE vms.pegawai SET nama=?,email=?,jabatan=NULLIF(?,''),kedeputian=NULLIF(?,''),direktorat=NULLIF(?,''),nomor_hp=NULLIF(?,''),nip=NULLIF(?,''),status_aktif=?,version=version+1
		WHERE id=? AND version=?`, after.Name, after.Email, after.JobTitle, after.Deputy, after.Directorate, after.Phone, after.NIP,
		after.Active, before.ID, before.Version)
	if result.Error != nil {
		return entity.Employee{}, fmt.Errorf("update employee: %w", employeeWriteError(result.Error))
	}
	if result.RowsAffected != 1 {
		return entity.Employee{}, errors.New("update employee: locked row or version changed")
	}
	after.Version = before.Version + 1
	if err := t.saveAccess(before, after); err != nil {
		return entity.Employee{}, err
	}
	return after, nil
}

func (t *transaction) saveAccess(before, after entity.Employee) error {
	if !slices.Equal(before.Roles, after.Roles) {
		if err := t.saveRoles(after.ID, after.Roles); err != nil {
			return err
		}
	}
	if slices.Equal(before.OfficerIDs, after.OfficerIDs) {
		return nil
	}
	if err := t.db.Exec("UPDATE vms.pegawai_pejabat SET status_aktif = FALSE WHERE id_pegawai = ? AND status_aktif", after.ID).Error; err != nil {
		return fmt.Errorf("retire PA assignment: %w", err)
	}
	if len(after.OfficerIDs) == 0 {
		return nil
	}
	if err := t.db.Exec(`INSERT INTO vms.pegawai_pejabat(id_pegawai,id_pejabat,status_aktif) VALUES (?,?,TRUE)
		ON CONFLICT (id_pegawai,id_pejabat) DO UPDATE SET status_aktif = TRUE`, after.ID, after.OfficerIDs[0]).Error; err != nil {
		return fmt.Errorf("store PA assignment: %w", err)
	}
	return nil
}

func (t *transaction) saveRoles(id int32, roles []string) error {
	if err := t.db.Exec(`DELETE FROM vms.role_pegawai a USING vms.role r
		WHERE a.id_karyawan = ? AND r.id = a.id_role AND r.nama NOT IN ?`, id, roles).Error; err != nil {
		return fmt.Errorf("remove employee roles: %w", err)
	}
	if err := t.db.Exec(`INSERT INTO vms.role_pegawai(id_karyawan,id_role)
		SELECT ?, id FROM vms.role WHERE nama IN ? ON CONFLICT (id_karyawan,id_role) DO NOTHING`,
		id, roles).Error; err != nil {
		return fmt.Errorf("store employee roles: %w", err)
	}
	return nil
}

func (t *transaction) EndSessions(employeeID int32, at time.Time) error {
	return t.db.Exec(`UPDATE support.staff_sessions SET revoked_at = ?, credential_generation = credential_generation + 1
		WHERE employee_id = ? AND revoked_at IS NULL`, at, employeeID).Error
}

func (t *transaction) ReloadSessions(employeeID int32, removedRoles []string, at time.Time) error {
	query := "UPDATE support.staff_sessions SET role_revision = role_revision + 1"
	args := []any{}
	if len(removedRoles) > 0 {
		query += ", active_role_id = CASE WHEN active_role_id IN (SELECT id FROM vms.role WHERE nama IN ?) THEN NULL ELSE active_role_id END"
		args = append(args, removedRoles)
	}
	return t.db.Exec(query+" WHERE employee_id = ? AND revoked_at IS NULL AND idle_expires_at > ? AND absolute_expires_at > ?",
		append(args, employeeID, at, at)...).Error
}

func auditValues(employee *entity.Employee) (any, error) {
	if employee == nil {
		return nil, nil
	}
	raw, err := json.Marshal(struct {
		Roles              []string `json:"roles"`
		Active             bool     `json:"active"`
		AssignedOfficerIDs []int32  `json:"assignedOfficerIds"`
		Version            int32    `json:"version"`
	}{employee.Roles, employee.Active, employee.OfficerIDs, employee.Version})
	return string(raw), err
}

func (t *transaction) Audit(event entity.EmployeeEvent) error {
	before, err := auditValues(event.Before)
	if err != nil {
		return err
	}
	after, err := auditValues(&event.After)
	if err != nil {
		return err
	}
	return t.db.Exec(`INSERT INTO vms.audit_log(actor_type,id_pegawai,entity_type,entity_id,jenis_kegiatan,tanggal,deskripsi,old_values,new_values)
		VALUES ('staff',?,'pegawai',?,?,?,?,CAST(? AS json),CAST(? AS json))`,
		event.Actor, event.After.ID, event.Action, event.At, event.Action, before, after).Error
}
