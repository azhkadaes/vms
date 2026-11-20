package authrepositoryimplementation

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/azhkadaes/vms/backend/internal/entity"
	authrepository "github.com/azhkadaes/vms/backend/internal/repository/auth"
	"gorm.io/gorm"
)

var _ authrepository.SessionRepository = (*repo)(nil)

func NewSessionRepository(db *gorm.DB) authrepository.SessionRepository { return &repo{db: db} }

func (r *repo) GetCredential(ctx context.Context, email string) (entity.EmployeeCredential, error) {
	var out entity.EmployeeCredential
	err := r.db.WithContext(ctx).Raw(`SELECT e.id AS employee_id,e.status_aktif AS active,c.encoded_hash FROM vms.pegawai e JOIN support.employee_credentials c ON c.employee_id=e.id WHERE e.email=?`, email).Scan(&out).Error
	return out, err
}
func (r *repo) FindRefresh(ctx context.Context, hash string) (string, error) {
	var id string
	err := r.db.WithContext(ctx).Raw(`SELECT id::text FROM support.staff_sessions WHERE refresh_hash=? UNION SELECT session_id::text FROM support.refresh_lineage WHERE token_hash=? LIMIT 1`, hash, hash).Scan(&id).Error
	if err == nil && id == "" {
		err = gorm.ErrRecordNotFound
	}
	return id, err
}
func admit(tx *gorm.DB, key string, limit int, window time.Duration) (int, int64, error) {
	if err := tx.Exec("INSERT INTO support.rate_limit_buckets(bucket_key) VALUES (?) ON CONFLICT DO NOTHING", key).Error; err != nil {
		return 0, 0, err
	}
	var locked string
	if err := tx.Raw("SELECT bucket_key FROM support.rate_limit_buckets WHERE bucket_key=? FOR UPDATE", key).Scan(&locked).Error; err != nil {
		return 0, 0, err
	}
	if locked != key {
		return 0, 0, errors.New("missing rate bucket")
	}
	var now time.Time
	if err := tx.Raw("SELECT clock_timestamp()").Scan(&now).Error; err != nil {
		return 0, 0, err
	}
	if err := tx.Exec("DELETE FROM support.rate_limit_events WHERE bucket_key=? AND occurred_at<=?", key, now.Add(-window)).Error; err != nil {
		return 0, 0, err
	}
	var history struct {
		Count  int
		Oldest *time.Time
	}
	if err := tx.Raw("SELECT count(*) AS count,min(occurred_at) AS oldest FROM support.rate_limit_events WHERE bucket_key=?", key).Scan(&history).Error; err != nil {
		return 0, 0, err
	}
	if history.Count >= limit {
		if history.Oldest == nil {
			return 0, 0, errors.New("invalid rate history")
		}
		return max(1, int(math.Ceil(history.Oldest.Add(window).Sub(now).Seconds()))), 0, nil
	}
	var id int64
	err := tx.Raw("INSERT INTO support.rate_limit_events(bucket_key,occurred_at) VALUES (?,?) RETURNING id", key, now).Scan(&id).Error
	return 0, id, err
}
func (r *repo) ReserveLogin(ctx context.Context, ipKey, accountKey string) (string, int, error) {
	var reservation int64
	var retry int
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		retry, _, err = admit(tx, ipKey, 100, 15*time.Minute)
		if err != nil || retry > 0 {
			return err
		}
		retry, reservation, err = admit(tx, accountKey, 5, 15*time.Minute)
		return err
	})
	if err != nil {
		return "", 0, err
	}
	return strconv.FormatInt(reservation, 10), retry, nil
}
func (r *repo) FinishLogin(ctx context.Context, reservation string, success bool) error {
	if !success {
		return nil
	}
	return r.db.WithContext(ctx).Exec("DELETE FROM support.rate_limit_events WHERE id=?", reservation).Error
}

type sessionTx struct {
	db  *gorm.DB
	now time.Time
}

func (r *repo) Transaction(ctx context.Context, employeeID int32, operation func(authrepository.SessionTransaction) error) error {
	return r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		var locked int32
		if err := db.Raw("SELECT id FROM vms.pegawai WHERE id=? FOR UPDATE", employeeID).Scan(&locked).Error; err != nil {
			return err
		}
		if locked != employeeID || locked <= 0 {
			return gorm.ErrRecordNotFound
		}
		var now time.Time
		if err := db.Raw("SELECT clock_timestamp()").Scan(&now).Error; err != nil {
			return err
		}
		return operation(&sessionTx{db: db, now: now})
	})
}
func (t *sessionTx) Now() time.Time { return t.now }
func (t *sessionTx) Credential(employeeID int32) (entity.EmployeeCredential, error) {
	var out entity.EmployeeCredential
	result := t.db.Raw(`SELECT e.id AS employee_id,e.status_aktif AS active,c.encoded_hash FROM vms.pegawai e JOIN support.employee_credentials c ON c.employee_id=e.id WHERE e.id=?`, employeeID).Scan(&out)
	if result.Error != nil {
		return out, result.Error
	}
	if result.RowsAffected != 1 {
		return out, gorm.ErrRecordNotFound
	}
	return out, nil
}
func (t *sessionTx) Roles(employeeID int32) (map[string]int32, error) {
	var rows []struct {
		ID   int32
		Name string
	}
	err := t.db.Raw(`SELECT r.id,r.nama AS name FROM vms.role_pegawai a JOIN vms.role r ON r.id=a.id_role WHERE a.id_karyawan=? ORDER BY r.id`, employeeID).Scan(&rows).Error
	out := map[string]int32{}
	for _, r := range rows {
		out[r.Name] = r.ID
	}
	return out, err
}
func (t *sessionTx) Session(id string) (entity.StaffSession, error) {
	var out entity.StaffSession
	result := t.db.Raw("SELECT * FROM support.staff_sessions WHERE id=? FOR UPDATE", id).Scan(&out)
	if result.Error != nil {
		return out, result.Error
	}
	if result.RowsAffected != 1 {
		return out, gorm.ErrRecordNotFound
	}
	return out, nil
}
func (t *sessionTx) Create(s entity.StaffSession) error {
	return t.db.Table("support.staff_sessions").Create(&s).Error
}
func (t *sessionTx) Save(s entity.StaffSession) error {
	result := t.db.Exec(`UPDATE support.staff_sessions SET active_role_id=?,credential_generation=?,role_revision=?,refresh_hash=?,last_activity_at=?,idle_expires_at=?,revoked_at=? WHERE id=?`, s.ActiveRoleID, s.CredentialGeneration, s.RoleRevision, s.RefreshHash, s.LastActivityAt, s.IdleExpiresAt, s.RevokedAt, s.ID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (t *sessionTx) SetPassword(employeeID int32, encoded, currentID string) error {
	if err := t.db.Exec("UPDATE support.employee_credentials SET encoded_hash=?,updated_at=? WHERE employee_id=?", encoded, t.now, employeeID).Error; err != nil {
		return err
	}
	if err := t.db.Exec("UPDATE vms.pegawai SET must_change_password=FALSE WHERE id=?", employeeID).Error; err != nil {
		return err
	}
	if err := t.db.Exec("UPDATE support.staff_sessions SET revoked_at=? WHERE employee_id=? AND id<>? AND revoked_at IS NULL", t.now, employeeID, currentID).Error; err != nil {
		return err
	}
	return t.db.Exec("DELETE FROM support.refresh_recovery WHERE session_id IN (SELECT id FROM support.staff_sessions WHERE employee_id=?)", employeeID).Error
}
func (t *sessionTx) Recovery(sessionID, requestID string) (entity.SessionRecovery, error) {
	var out entity.SessionRecovery
	result := t.db.Raw("SELECT * FROM support.refresh_recovery WHERE session_id=? AND request_id=?", sessionID, requestID).Scan(&out)
	if result.Error != nil {
		return out, result.Error
	}
	if result.RowsAffected != 1 {
		return out, gorm.ErrRecordNotFound
	}
	return out, nil
}
func (t *sessionTx) SaveRecovery(r entity.SessionRecovery) error {
	return t.db.Table("support.refresh_recovery").Create(&r).Error
}
func (t *sessionTx) ClearRecoveries(id string) error {
	return t.db.Exec("DELETE FROM support.refresh_recovery WHERE session_id=?", id).Error
}
func (t *sessionTx) RememberRefresh(id, hash string) error {
	return t.db.Exec("INSERT INTO support.refresh_lineage(token_hash,session_id,consumed_at) VALUES (?,?,?) ON CONFLICT DO NOTHING", hash, id, t.now).Error
}

// Audit records a session lifecycle event: the staff member is both the actor and the pegawai target.
func (t *sessionTx) Audit(id int32, action string) error {
	err := t.db.Exec(`INSERT INTO vms.audit_log(actor_type,id_pegawai,entity_type,entity_id,jenis_kegiatan,tanggal,deskripsi)
		VALUES ('staff',?,'pegawai',?,?,?,?)`, id, id, action, t.now, fmt.Sprintf("Staff session action: %s", action)).Error
	if err != nil {
		return fmt.Errorf("audit session action: %w", err)
	}
	return nil
}
