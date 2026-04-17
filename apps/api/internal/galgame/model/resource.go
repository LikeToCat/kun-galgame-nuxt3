package model

// GalgameResourceRow is a flat projection of galgame_resource used for reads.
// It doesn't drive migrations; it's the shape that repository queries return.
type GalgameResourceRow struct {
	ID        int     `gorm:"column:id"`
	View      int     `gorm:"column:view"`
	GalgameID int     `gorm:"column:galgame_id"`
	UserID    int     `gorm:"column:user_id"`
	Type      string  `gorm:"column:type"`
	Language  string  `gorm:"column:language"`
	Platform  string  `gorm:"column:platform"`
	Size      string  `gorm:"column:size"`
	Status    int     `gorm:"column:status"`
	Download  int     `gorm:"column:download"`
	LikeCount int     `gorm:"column:like_count"`
	Code      string  `gorm:"column:code"`
	Password  string  `gorm:"column:password"`
	Note      string  `gorm:"column:note"`
	Created   string  `gorm:"column:created"`
	Edited    *string `gorm:"column:edited"`
}

// ResourceAggregate is used when aggregating DISTINCT platform/language/type
// per galgame.
type ResourceAggregate struct {
	Platform string `gorm:"column:platform"`
	Language string `gorm:"column:language"`
	Type     string `gorm:"column:type"`
}
