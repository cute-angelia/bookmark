package model

// BookmarkModel is the record for an URL.
type BookmarkModel struct {
	ID         int        `gorm:"column:id"  db:"id"            json:"id"`
	URL        string     `gorm:"column:url"  db:"url"           json:"url"`
	Title      string     `gorm:"column:title"  db:"title"         json:"title"`
	ImageURL   string     `gorm:"column:image_url"      db:"image_url"         json:"image_url"`
	Excerpt    string     `gorm:"column:excerpt"  db:"excerpt"       json:"excerpt"`
	Uid        int        `gorm:"column:uid"  db:"uid"        json:"uid"`
	Tags       string     `gorm:"column:tags"  db:"tags"        json:"tags"`
	Public     int        `gorm:"column:public"  db:"public"        json:"public"`
	Modified   string     `gorm:"column:modified"  db:"modified"      json:"modified"`
	TagsDetail []TagModel `json:"tags_detail"  db:"-"     gorm:"-"`
}

func (BookmarkModel) TableName() string {
	return "bookmark"
}
