package internal

import (
	model2 "bookmark/cmd/bookmark/model"
	"github.com/cute-angelia/go-utils/components/igorm"
	"strings"
)

type tagsInternal struct {
}

func NewTagInternal() *tagsInternal {
	return &tagsInternal{}
}

func (that tagsInternal) GetTags(tagIds string) (tagModels []model2.TagModel) {
	orm, _ := igorm.GetGormSQLite("cache")
	tags := []model2.TagModel{}

	querytags := strings.Split(tagIds, ",")
	for _, querytag := range querytags {
		itag := model2.TagModel{}
		orm.Where("id = ?", querytag).First(&itag)
		tags = append(tags, itag)
	}

	return tags
}

func (that tagsInternal) InsertTag(tags []string) (tagModels []model2.TagModel, tagIds []int) {
	orm, _ := igorm.GetGormSQLite("cache")
	for _, name := range tags {
		tagmodel := model2.TagModel{
			Name: strings.TrimSpace(name),
		}
		orm.Where("name = ?", name).FirstOrCreate(&tagmodel)
		tagModels = append(tagModels, tagmodel)
		tagIds = append(tagIds, tagmodel.ID)
	}
	return
}

func (that tagsInternal) UpdateRelationship(bookmarkId int, tagIds []int) {
	orm, _ := igorm.GetGormSQLite("cache")
	// 删除之前关系
	orm.Table(model2.BookmarkTagModel{}.TableName()).Where("bookmark_id = ?", bookmarkId).Delete(model2.BookmarkTagModel{})

	for _, id := range tagIds {
		modelBt := model2.BookmarkTagModel{
			BookmarkId: bookmarkId,
			TagId:      id,
		}
		orm.Where("bookmark_id = ? and tag_id = ?", bookmarkId, id).FirstOrCreate(&modelBt)
	}
}
