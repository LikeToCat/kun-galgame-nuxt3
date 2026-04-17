package repository

import "gorm.io/gorm"

// GalgameDetailRatingRepository serves the ratings block on the galgame
// detail page (joined rows + the user's liked-rating set).
//
// NOTE: full CRUD for ratings lives in RatingRepository (rating_repo.go);
// this repo is read-only for the detail page.
type GalgameDetailRatingRepository struct {
	db *gorm.DB
}

func NewGalgameDetailRatingRepository(db *gorm.DB) *GalgameDetailRatingRepository {
	return &GalgameDetailRatingRepository{db: db}
}

func (r *GalgameDetailRatingRepository) DB() *gorm.DB { return r.db }

// GalgameDetailRatingRow is a joined rating row used by galgame detail.
type GalgameDetailRatingRow struct {
	ID           int    `gorm:"column:id"`
	Recommend    string `gorm:"column:recommend"`
	Overall      int    `gorm:"column:overall"`
	View         int    `gorm:"column:view"`
	GalgameType  string `gorm:"column:galgame_type"`
	PlayStatus   string `gorm:"column:play_status"`
	ShortSummary string `gorm:"column:short_summary"`
	SpoilerLevel string `gorm:"column:spoiler_level"`
	Art          int    `gorm:"column:art"`
	Story        int    `gorm:"column:story"`
	Music        int    `gorm:"column:music"`
	Character    int    `gorm:"column:character"`
	Route        int    `gorm:"column:route"`
	System       int    `gorm:"column:system"`
	Voice        int    `gorm:"column:voice"`
	ReplayValue  int    `gorm:"column:replay_value"`
	LikeCount    int    `gorm:"column:like_count"`
	Created      string `gorm:"column:created"`
	Updated      string `gorm:"column:updated"`
	UserID       int    `gorm:"column:user_id"`
}

// FindRatingsByGalgame returns all ratings for a galgame, ordered by created DESC.
func (r *GalgameDetailRatingRepository) FindRatingsByGalgame(galgameID int) []GalgameDetailRatingRow {
	var rows []GalgameDetailRatingRow
	r.db.Table("galgame_rating").
		Where("galgame_id = ?", galgameID).
		Order("created DESC").
		Scan(&rows)
	return rows
}

// FindLikedRatingIDs returns the set of rating IDs the user has liked, from a
// given candidate list.
func (r *GalgameDetailRatingRepository) FindLikedRatingIDs(userID int, ratingIDs []int) map[int]bool {
	out := map[int]bool{}
	if userID <= 0 || len(ratingIDs) == 0 {
		return out
	}
	type row struct {
		GalgameRatingID int `gorm:"column:galgame_rating_id"`
	}
	var rows []row
	r.db.Table("galgame_rating_like").
		Select("galgame_rating_id").
		Where("user_id = ? AND galgame_rating_id IN ?", userID, ratingIDs).
		Scan(&rows)
	for _, x := range rows {
		out[x.GalgameRatingID] = true
	}
	return out
}
