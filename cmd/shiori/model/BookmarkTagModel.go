package model

type BookmarkTagModel struct {
	BookmarkId int `gorm:"column:bookmark_id"  db:"bookmark_id"            json:"bookmark_id"`
	TagId      int `gorm:"column:tag_id"  db:"tag_id"           json:"tag_id"`
}

func (BookmarkTagModel) TableName() string {
	return "bookmark_tag"
}
