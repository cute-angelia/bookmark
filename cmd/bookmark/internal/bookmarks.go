package internal

import (
	"bookmark/cmd/bookmark/database"
	"bookmark/cmd/bookmark/internal/consts"
	model2 "bookmark/cmd/bookmark/model"
	"bookmark/pkg/db"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/cute-angelia/go-utils/components/igorm"
	"github.com/guonaihong/gout"
	"gorm.io/gorm"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type bookmarksInternal struct {
	orm *gorm.DB
}

func NewBookmarksInternal() *bookmarksInternal {
	orm, _ := igorm.GetGormSQLite("cache")
	return &bookmarksInternal{
		orm: orm,
	}
}

func (that bookmarksInternal) Info(uri string) model2.BookmarkModel {
	bookmark := model2.BookmarkModel{}
	that.orm.Where("url = ?", uri).First(&bookmark)
	return bookmark
}

// GetBookmarkList 查询书签列表
func (that bookmarksInternal) GetBookmarkList(opts database.GetBookmarksOptions, page, perpage int) (list []model2.BookmarkModel, count int64) {
	ormSearch := that.orm

	ormSearch = ormSearch.Order("id desc")

	// Add where clause for IDs
	if len(opts.IDs) > 0 {
		ormSearch = ormSearch.Or("id in (?)", opts.IDs)
	}

	if len(opts.Keyword) > 0 {
		ormSearch = ormSearch.Where("title like ? ", "%"+opts.Keyword+"%")
	}

	if len(opts.Tags) > 0 {
		tags := []model2.TagModel{}
		that.orm.Where("name in (?)", opts.Tags).Find(&tags)
		for _, tag := range tags {
			ormSearch = ormSearch.Where("tags like ? ", fmt.Sprint("%"+fmt.Sprintf("%d", tag.ID)+"%"))
		}
	}

	list, count, _ = db.Paginate[model2.BookmarkModel](ormSearch, page, perpage)

	// 组装 tags_detail
	for i, model := range list {
		tagsarray := strings.Split(model.Tags, ",")
		for _, tagIdstr := range tagsarray {
			tagId, _ := strconv.Atoi(tagIdstr)
			if tagId > 0 {
				list[i].TagsDetail = append(list[i].TagsDetail, that.GetTagInfo(tagId))
			}
		}
	}

	return
}

// GetTagInfo 获取tag信息
func (that bookmarksInternal) GetTagInfo(tagId int) (tag model2.TagModel) {
	that.orm.Where("id = ?", tagId).First(&tag)
	return
}

// GetTagIdByBookMarkId 书签获取 tagId
func (that bookmarksInternal) GetTagIdByBookMarkId(id int) (ids []int) {
	bookmarkTags := []model2.BookmarkTagModel{}
	that.orm.Where("bookmark_id = ?", id).Find(&bookmarkTags)

	for _, tag := range bookmarkTags {
		ids = append(ids, tag.TagId)
	}
	return
}

func (that bookmarksInternal) InfoById(id int) model2.BookmarkModel {
	orm, _ := igorm.GetGormSQLite("cache")
	bookmark := model2.BookmarkModel{}
	orm.Where("id = ?", id).First(&bookmark)
	return bookmark
}

func (that bookmarksInternal) Delete(bookmark model2.BookmarkModel) {
	orm, _ := igorm.GetGormSQLite("cache")
	// 删除之前关系
	orm.Table(model2.BookmarkTagModel{}.TableName()).Where("bookmark_id = ?", bookmark.ID).Delete(model2.BookmarkTagModel{})
	orm.Table(model2.BookmarkModel{}.TableName()).Where("id = ?", bookmark.ID).Delete(model2.BookmarkModel{})

	// 删除缩略图
	os.Remove(filepath.Join(consts.UploadDir, bookmark.ImageURL))
}

// CreateOrEdit 创建书签获取更新书签
func (that bookmarksInternal) CreateOrEdit(bookmark model2.BookmarkModel, tags []string) model2.BookmarkModel {
	orm, _ := igorm.GetGormSQLite("cache")

	if len(bookmark.Title) == 0 {
		bookmark = that.GetNetInfo(bookmark)
	}

	// 判断是更新还是新建
	if bookmark.ID > 0 {
		// 更新标签关系
		tagInternal := NewTagInternal()
		_, tagIds := tagInternal.InsertTag(tags)
		tagInternal.UpdateRelationship(bookmark.ID, tagIds)

		var idStrings []string
		for _, id := range tagIds {
			idStrings = append(idStrings, strconv.Itoa(id))
		}
		bookmark.Tags = strings.Join(idStrings, ",")

		// 更新书签内容
		orm.Save(&bookmark)

		//go that.CatchShotPicture(bookmark)
	} else {
		// 标签处理
		tagInternal := NewTagInternal()
		_, tagIds := tagInternal.InsertTag(tags)

		// 新建书签
		var idStrings []string
		for _, id := range tagIds {
			idStrings = append(idStrings, strconv.Itoa(id))
		}
		bookmark.Tags = strings.Join(idStrings, ",")
		orm.Create(&bookmark)

		// 更新标签关系
		tagInternal.UpdateRelationship(bookmark.ID, tagIds)

		//go that.CatchShotPicture(bookmark)
	}

	return bookmark
}

// 抓取缩略图
func (that bookmarksInternal) CatchShotPicture(bookmark model2.BookmarkModel) {
	// todo
	log.Print("抓取缩略图")
}

func (that bookmarksInternal) GetNetInfo(bookmark model2.BookmarkModel) model2.BookmarkModel {
	resp := that.getContent(bookmark.URL)
	if q, err := goquery.NewDocumentFromReader(strings.NewReader(resp)); err != nil {
		log.Println("goquery.new", err)
	} else {
		title := q.Find("title").Text()
		if len(title) > 0 {
			bookmark.Title = title
		}
	}
	return bookmark
}

func (that bookmarksInternal) getContent(uri string) string {
	var resp string
	if err := gout.GET(uri).SetHeader(gout.H{
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36",
	}).SetTimeout(time.Second * 10).BindBody(&resp).Do(); err != nil {
		proxySocks5 := strings.Replace(os.Getenv("PROXYADDR"), "socks5://", "", -1)
		gout.GET(uri).SetHeader(gout.H{
			"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36",
		}).SetSOCKS5(proxySocks5).SetTimeout(time.Second * 10).BindBody(&resp).Do()
	}
	return resp
}
