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

// NewCatalogRepository binds public-only catalog queries to the supplied database pool.
func NewCatalogRepository(db *gorm.DB) catalogrepository.CatalogRepository { return &repo{db: db} }

// GetPublicCatalogs derives VIP through Jabatan inside a read-only repeatable-read transaction.
func (r *repo) GetPublicCatalogs(ctx context.Context) (entity.PublicCatalog, error) {
	result := entity.PublicCatalog{Officers: []entity.CatalogOfficer{}, Rooms: []entity.CatalogRoom{}}
	var officers []officerRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("vms.pejabat AS p").
			Select("p.id, p.nama AS name, j.pejabat_vip AS vip, p.status_aktif AS active, j.nama AS job_title, COALESCE(d.nama, '') AS directorate, COALESCE(k.nama, '') AS deputy").
			Joins("JOIN vms.jabatan AS j ON j.id = p.id_jabatan").
			Joins("LEFT JOIN vms.direktorat AS d ON d.id = j.id_direktorat").
			Joins("LEFT JOIN vms.kedeputian AS k ON k.id = COALESCE(d.id_kedeputian, j.id_kedeputian)").
			Where("p.status_aktif = TRUE").Order("p.nama ASC, p.id ASC").Find(&officers).Error; err != nil {
			return err
		}
		return tx.Table("vms.ruangan").Select("id, nama AS name, status_aktif AS active").Where("status_aktif = TRUE").Order("nama ASC, id ASC").Find(&result.Rooms).Error
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return entity.PublicCatalog{}, fmt.Errorf("read public catalog snapshot: %w", err)
	}
	for _, row := range officers {
		result.Officers = append(result.Officers, entity.CatalogOfficer{ID: row.ID, Name: row.Name, VIP: row.VIP, Active: row.Active, JobTitle: row.JobTitle, Directorate: row.Directorate, Deputy: row.Deputy})
	}
	return result, nil
}
