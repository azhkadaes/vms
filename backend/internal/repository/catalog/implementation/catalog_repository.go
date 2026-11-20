package catalogrepositoryimplementation

import (
	"context"
	"database/sql"
	"fmt"

	"gorm.io/gorm"

	"github.com/azhkadaes/vms/backend/internal/entity"
	catalogrepository "github.com/azhkadaes/vms/backend/internal/repository/catalog"
)

type repo struct{ db *gorm.DB }

var _ catalogrepository.CatalogRepository = (*repo)(nil)

// VIP is not a GORM-recognized initialism; bind the SQL alias explicitly.
type officerRow struct {
	ID          int32
	Name        string
	VIP         bool `gorm:"column:vip"`
	Active      bool
	JobTitle    string
	Directorate string
	Deputy      string
}

// NewCatalogRepository binds the catalog queries to the supplied database pool.
func NewCatalogRepository(db *gorm.DB) catalogrepository.CatalogRepository { return &repo{db: db} }

// activeOfficers is every scope's officer filter.
const activeOfficers = "p.status_aktif = TRUE"

// assignedOfficers adds inactive officers an employee still holds an active assignment to, whatever
// that employee's own status: the editor also opens deactivated employees. EXISTS lists each once.
const assignedOfficers = activeOfficers + " OR EXISTS (SELECT 1 FROM vms.pegawai_pejabat AS a WHERE a.id_pejabat = p.id AND a.status_aktif = TRUE)"

// GetPublicCatalogs derives VIP through Jabatan inside a read-only repeatable-read transaction.
func (r *repo) GetPublicCatalogs(ctx context.Context) (entity.PublicCatalog, error) {
	officers, rooms, err := r.destinations(ctx, activeOfficers)
	if err != nil {
		return entity.PublicCatalog{}, fmt.Errorf("read public catalog snapshot: %w", err)
	}
	return entity.PublicCatalog{Officers: officers, Rooms: rooms}, nil
}

// GetStaffCatalogs reads the same officer and room shapes as the public catalog in one snapshot.
func (r *repo) GetStaffCatalogs(ctx context.Context, assignedInactiveOfficers bool) (entity.StaffCatalog, error) {
	filter := activeOfficers
	if assignedInactiveOfficers {
		filter = assignedOfficers
	}
	officers, rooms, err := r.destinations(ctx, filter)
	if err != nil {
		return entity.StaffCatalog{}, fmt.Errorf("read staff catalog snapshot: %w", err)
	}
	return entity.StaffCatalog{Officers: officers, Rooms: rooms}, nil
}

// destinations reads the officers matching officerFilter and the active rooms in one read-only
// repeatable-read transaction. Neither list is nil.
func (r *repo) destinations(ctx context.Context, officerFilter string) ([]entity.CatalogOfficer, []entity.CatalogRoom, error) {
	var officers []officerRow
	rooms := []entity.CatalogRoom{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("vms.pejabat AS p").
			Select("p.id, p.nama AS name, j.pejabat_vip AS vip, p.status_aktif AS active, j.nama AS job_title, COALESCE(d.nama, '') AS directorate, COALESCE(k.nama, '') AS deputy").
			Joins("JOIN vms.jabatan AS j ON j.id = p.id_jabatan").
			Joins("LEFT JOIN vms.direktorat AS d ON d.id = j.id_direktorat").
			Joins("LEFT JOIN vms.kedeputian AS k ON k.id = COALESCE(d.id_kedeputian, j.id_kedeputian)").
			Where(officerFilter).Order("p.nama ASC, p.id ASC").Find(&officers).Error; err != nil {
			return err
		}
		return tx.Table("vms.ruangan").Select("id, nama AS name, status_aktif AS active").Where("status_aktif = TRUE").Order("nama ASC, id ASC").Find(&rooms).Error
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, nil, err
	}
	out := make([]entity.CatalogOfficer, 0, len(officers))
	for _, row := range officers {
		out = append(out, entity.CatalogOfficer{ID: row.ID, Name: row.Name, VIP: row.VIP, Active: row.Active, JobTitle: row.JobTitle, Directorate: row.Directorate, Deputy: row.Deputy})
	}
	return out, rooms, nil
}
