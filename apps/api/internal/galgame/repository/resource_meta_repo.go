package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

// GalgameResourceMetaRepository owns DISTINCT aggregates across
// galgame_resource (platform/language/type) for detail pages and list cards.
type GalgameResourceMetaRepository struct {
	db *gorm.DB
}

func NewGalgameResourceMetaRepository(db *gorm.DB) *GalgameResourceMetaRepository {
	return &GalgameResourceMetaRepository{db: db}
}

func (r *GalgameResourceMetaRepository) DB() *gorm.DB { return r.db }

// FindResourceMetaByGalgame aggregates DISTINCT platform/language/type for one galgame.
func (r *GalgameResourceMetaRepository) FindResourceMetaByGalgame(galgameID int) (platforms, languages, types []string) {
	type row struct {
		Platform string `gorm:"column:platform"`
		Language string `gorm:"column:language"`
		Type     string `gorm:"column:type"`
	}
	var rows []row
	r.db.Table("galgame_resource").
		Select("DISTINCT platform, language, type").
		Where("galgame_id = ?", galgameID).Scan(&rows)

	pSet, lSet, tSet := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, x := range rows {
		if x.Platform != "" {
			pSet[x.Platform] = true
		}
		if x.Language != "" {
			lSet[x.Language] = true
		}
		if x.Type != "" {
			tSet[x.Type] = true
		}
	}
	return mapKeys(pSet), mapKeys(lSet), mapKeys(tSet)
}

// FindResourceMetaBatch aggregates DISTINCT (galgame_id, platform, language)
// tuples for a batch of galgame IDs.
func (r *GalgameResourceMetaRepository) FindResourceMetaBatch(galgameIDs []int) []model.GalgameResourceMeta {
	if len(galgameIDs) == 0 {
		return nil
	}
	var rows []model.GalgameResourceMeta
	r.db.Table("galgame_resource").
		Select("DISTINCT galgame_id, platform, language").
		Where("galgame_id IN ?", galgameIDs).Scan(&rows)
	return rows
}

func mapKeys(m map[string]bool) []string {
	if m == nil {
		return []string{}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
