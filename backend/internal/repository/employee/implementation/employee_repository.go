package employeerepositoryimplementation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	employeerepository "github.com/azhkadaes/vms/backend/internal/repository/employee"
)

type repository struct{ db *gorm.DB }

var _ employeerepository.EmployeeRepository = (*repository)(nil)

func NewEmployeeRepository(db *gorm.DB) employeerepository.EmployeeRepository {
	return &repository{db: db}
}

const employeeSelect = `SELECT id, nama AS name, email, status_aktif AS active,
	must_change_password, version, COALESCE(jabatan, '') AS job_title, COALESCE(kedeputian, '') AS deputy,
	COALESCE(direktorat, '') AS directorate, array_to_json(nomor_hp)::text AS phone_numbers, COALESCE(nip, '') AS nip FROM vms.pegawai`

// employeeRow keeps association slices out of GORM's scan target.
type employeeRow struct {
	ID                 int32
	Name               string
	Email              string
	JobTitle           string
	Deputy             string
	Directorate        string
	PhoneNumbers       string
	NIP                string
	Active             bool
	MustChangePassword bool
	Version            int32
}

func withAccess(db *gorm.DB, rows []employeeRow) ([]entity.Employee, error) {
	out := make([]entity.Employee, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]int32, len(rows))
	index := make(map[int32]int, len(rows))
	for i, row := range rows {
		ids[i], index[row.ID] = row.ID, i
		var phoneNumbers []string
		if err := json.Unmarshal([]byte(row.PhoneNumbers), &phoneNumbers); err != nil {
			return nil, fmt.Errorf("read employee phone numbers: %w", err)
		}
		out[i] = entity.Employee{
			ID: row.ID, Name: row.Name, Email: row.Email, JobTitle: row.JobTitle, Deputy: row.Deputy,
			Directorate: row.Directorate, PhoneNumbers: phoneNumbers, NIP: row.NIP, Active: row.Active,
			MustChangePassword: row.MustChangePassword, Version: row.Version, Roles: []string{}, OfficerIDs: []int32{},
		}
	}
	var roles []struct {
		EmployeeID int32
		Role       string
	}
	if err := db.Raw(`SELECT a.id_karyawan AS employee_id, r.nama AS role
		FROM vms.role_pegawai a JOIN vms.role r ON r.id = a.id_role
		WHERE a.id_karyawan IN ? ORDER BY r.id`, ids).Scan(&roles).Error; err != nil {
		return nil, fmt.Errorf("read employee roles: %w", err)
	}
	for _, role := range roles {
		i := index[role.EmployeeID]
		out[i].Roles = append(out[i].Roles, role.Role)
	}
	var officers []struct {
		EmployeeID int32
		OfficerID  int32
	}
	if err := db.Raw(`SELECT id_pegawai AS employee_id, id_pejabat AS officer_id
		FROM vms.pegawai_pejabat WHERE status_aktif AND id_pegawai IN ? ORDER BY id_pejabat`, ids).Scan(&officers).Error; err != nil {
		return nil, fmt.Errorf("read employee assignments: %w", err)
	}
	for _, officer := range officers {
		i := index[officer.EmployeeID]
		out[i].OfficerIDs = append(out[i].OfficerIDs, officer.OfficerID)
	}
	return out, nil
}

func (r *repository) GetEmployees(ctx context.Context, filter entity.EmployeeFilter) (entity.EmployeePage, error) {
	var page entity.EmployeePage
	opts := &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		const count = "SELECT count(*) FROM vms.pegawai"
		if err := tx.Raw(count).Scan(&page.TotalAll).Error; err != nil {
			return fmt.Errorf("count employees: %w", err)
		}
		var where string
		var args []any
		page.Total = page.TotalAll
		if filter.Query != "" {
			where = ` WHERE strpos(lower(nama), lower(?)) > 0 OR strpos(lower(email), lower(?)) > 0
				OR strpos(lower(COALESCE(nip, '')), lower(?)) > 0`
			args = []any{filter.Query, filter.Query, filter.Query}
			if err := tx.Raw(count+where, args...).Scan(&page.Total).Error; err != nil {
				return fmt.Errorf("count matching employees: %w", err)
			}
		}
		offset := (int64(filter.Page) - 1) * int64(filter.PageSize)
		var rows []employeeRow
		err := tx.Raw(employeeSelect+where+" ORDER BY nama, id LIMIT ? OFFSET ?",
			append(args, filter.PageSize, offset)...).Scan(&rows).Error
		if err != nil {
			return fmt.Errorf("list employees: %w", err)
		}
		page.Items, err = withAccess(tx, rows)
		return err
	}, opts)
	return page, err
}

func (r *repository) GetEmployee(ctx context.Context, id int32) (entity.Employee, error) {
	var item entity.Employee
	opts := &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		item, err = getEmployee(tx, id, false)
		return err
	}, opts)
	return item, err
}

func getEmployee(db *gorm.DB, id int32, lock bool) (entity.Employee, error) {
	query := employeeSelect + " WHERE id = ?"
	if lock {
		query += " FOR UPDATE"
	}
	var rows []employeeRow
	if err := db.Raw(query, id).Scan(&rows).Error; err != nil {
		return entity.Employee{}, fmt.Errorf("read employee: %w", err)
	}
	if len(rows) == 0 {
		return entity.Employee{}, employeerepository.ErrNotFound
	}
	items, err := withAccess(db, rows)
	if err != nil {
		return entity.Employee{}, err
	}
	return items[0], nil
}
