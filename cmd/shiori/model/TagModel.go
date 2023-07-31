package model

// TagModel is the tag for a bookmark.
type TagModel struct {
	ID   int    `db:"id"          gorm:"column:id"          json:"id"`
	Name string `db:"name"        gorm:"column:name"        json:"name"`
}

func (TagModel) TableName() string {
	return "tag"
}
