package internal

import (
	"bookmark/cmd/bookmark/internal/consts"
	model2 "bookmark/cmd/bookmark/model"
	"github.com/PuerkitoBio/goquery"
	"github.com/cute-angelia/go-utils/components/igorm"
	"github.com/guonaihong/gout"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type bookmarksInternal struct {
}

func NewBookmarksInternal() *bookmarksInternal {
	return &bookmarksInternal{}
}
func (that bookmarksInternal) Info(uri string) model2.BookmarkModel {
	orm, _ := igorm.GetGormSQLite("cache")
	bookmark := model2.BookmarkModel{}
	orm.Where("url = ?", uri).First(&bookmark)
	return bookmark
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
