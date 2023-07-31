package bookmarks

import (
	"bookmark/cmd/shiori/database"
	"bookmark/cmd/shiori/internal"
	"bookmark/pkg/utils"
	"context"
	"github.com/cute-angelia/go-utils/syntax/itime"
	"github.com/cute-angelia/go-utils/utils/http/api"
	"github.com/cute-angelia/go-utils/utils/http/apiV2"
	"github.com/cute-angelia/go-utils/utils/http/validation"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"math"
	"net/http"
	"strings"
)

type Bookmarks struct {
}

func (that Bookmarks) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", that.lists)

	r.Post("/add", that.add)

	r.Post("/delete", that.delete)
	return r
}

func (that Bookmarks) lists(w http.ResponseWriter, r *http.Request) {
	// 校验参数
	valid := validation.Validation{}
	body := apiV2.NewBody(r)
	u := struct {
		Keyword         string
		StrPage         int32
		StrTags         string
		StrExcludedTags string
	}{
		Keyword:         body.PostString("keyword"),
		StrPage:         body.PostInt32("page"),
		StrTags:         body.PostString("tags"),
		StrExcludedTags: body.PostString("exclude"),
	}
	if err := valid.Submit(u); err != nil {
		api.Error(w, r, nil, err.Error(), -1)
		return
	}

	tags := strings.Split(u.StrTags, ",")
	if len(tags) == 1 && tags[0] == "" {
		tags = []string{}
	}

	excludedTags := strings.Split(u.StrExcludedTags, ",")
	if len(excludedTags) == 1 && excludedTags[0] == "" {
		excludedTags = []string{}
	}

	page := u.StrPage
	if page < 1 {
		page = 1
	}

	offset := (int(page) - 1) * 30

	// Prepare filter for database
	searchOptions := database.GetBookmarksOptions{
		Tags:         tags,
		ExcludedTags: excludedTags,
		Keyword:      u.Keyword,
		Limit:        30,
		Offset:       offset,
		OrderMethod:  database.ByLastAdded,
	}
	ctx := context.Background()
	// Calculate max page
	if nBookmarks, err := database.Dbx.GetBookmarksCount(ctx, searchOptions); err != nil {
		api.Error(w, r, nil, err.Error(), -1)
		return
	} else {
		maxPage := int(math.Ceil(float64(nBookmarks) / 30))
		// Fetch all matching bookmarks
		if bookmarks, err := database.Dbx.GetBookmarks(ctx, searchOptions); err != nil {
			api.Error(w, r, nil, err.Error(), -1)
			return
		} else {
			// Return JSON response
			resp := map[string]interface{}{
				"page":      page,
				"maxPage":   maxPage,
				"bookmarks": bookmarks,
			}
			api.Success(w, r, resp, "获取书签列表")
			return
		}
	}
}

func (that Bookmarks) add(w http.ResponseWriter, r *http.Request) {
	// 校验参数
	valid := validation.Validation{}
	var u = struct {
		Url      string   `json:"url" valid:"Required;"`
		Title    string   `json:"title"`
		Excerpt  string   `json:"excerpt"`
		Public   int32    `json:"public"`
		Tags     []string `json:"tags"`
		LoginUid int32
	}{
		LoginUid: apiV2.GetLoginUid(r),
	}
	// 绑定数据
	apiV2.Bind(r, &u)

	if err := valid.Submit(u); err != nil {
		apiV2.Error(w, r, err)
		return
	}

	var err error
	if u.Url, err = utils.RemoveUTMParams(u.Url); err != nil {
		apiV2.Error(w, r, err)
		return
	}

	bookmarkInternal := internal.NewBookmarksInternal()
	bookmark := bookmarkInternal.Info(u.Url)

	if bookmark.ID <= 0 {
		bookmark.URL = u.Url
		bookmark.Title = u.Title
		bookmark.Excerpt = u.Excerpt
		bookmark.Public = int(u.Public)
		bookmark.Uid = int(u.LoginUid)
		bookmark.Modified = itime.NewUnixNow().Format()
	}

	book := bookmarkInternal.CreateOrEdit(bookmark, u.Tags)

	apiV2.Success(w, r, book, "添加书签成功")
	return
}

func (that Bookmarks) delete(w http.ResponseWriter, r *http.Request) {
	// 校验参数
	valid := validation.Validation{}
	body := apiV2.NewBody(r)
	u := struct {
		Id  int32 `valid:"Required;"`
		Uid int
	}{
		Id:  body.PostInt32("id"),
		Uid: int(apiV2.GetLoginUid(r)),
	}
	if err := valid.Submit(u); err != nil {
		api.Error(w, r, nil, err.Error(), -1)
		return
	}

	bookmarkInternal := internal.NewBookmarksInternal()
	bookmark := bookmarkInternal.InfoById(int(u.Id))

	if bookmark.ID <= 0 {
		apiV2.Error(w, r, errors.New("书签不存在"))
		return
	} else {
		bookmarkInternal.Delete(bookmark)
	}

	apiV2.Success(w, r, nil, "删除成功")
	return
}
