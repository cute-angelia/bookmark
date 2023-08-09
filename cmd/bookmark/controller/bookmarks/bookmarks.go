package bookmarks

import (
	"bookmark/cmd/bookmark/database"
	"bookmark/cmd/bookmark/internal"
	"bookmark/cmd/bookmark/internal/consts"
	"bookmark/cmd/bookmark/internal/errorcode"
	"bookmark/pkg/utils"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/cute-angelia/go-utils/syntax/itime"
	"github.com/cute-angelia/go-utils/utils/http/api"
	"github.com/cute-angelia/go-utils/utils/http/apiV2"
	"github.com/cute-angelia/go-utils/utils/http/validation"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"image"
	"image/jpeg"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Bookmarks struct {
}

func (that Bookmarks) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", that.lists)

	r.Post("/add", that.add)

	r.Post("/delete", that.delete)
	r.Post("/deleteUrl", that.deleteByUrl)

	r.Get("/showShot", that.showShot)
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
		Url       string `json:"url" valid:"Required;"`
		Title     string `json:"title"`
		From      string `json:"from"` // 来源 ext：插件
		Excerpt   string `json:"excerpt"`
		Public    int32  `json:"public"`
		Imgbase64 string `json:"imgbase64"`
		Tags      string `json:"tags"`
		LoginUid  int32
	}{
		LoginUid: apiV2.GetLoginUid(r),
	}
	// 绑定数据
	apiV2.Bind(r, &u)

	if err := valid.Submit(u); err != nil {
		apiV2.Error(w, r, err)
		return
	}

	tags := []string{}
	if len(u.Tags) > 0 {
		if strings.Contains(u.Tags, ",") {
			tags = strings.Split(u.Tags, ",")
		} else {
			tags = strings.Split(u.Tags, " ")
		}
	}

	var err error
	if u.Url, err = utils.RemoveUTMParams(u.Url); err != nil {
		apiV2.Error(w, r, err)
		return
	}

	bookmarkInternal := internal.NewBookmarksInternal()
	bookmark := bookmarkInternal.Info(u.Url)

	bookmark.Tags = strings.Join(tags, ",")
	bookmark.URL = u.Url
	bookmark.Title = u.Title
	bookmark.Excerpt = u.Excerpt
	bookmark.Public = int(u.Public)
	bookmark.Uid = int(u.LoginUid)
	bookmark.Modified = itime.NewUnixNow().Format()

	if bookmark.ID <= 0 {
		// 保存图片
		if bookmark.ImageURL, err = that.saveImageFromBase64(u.Imgbase64); err != nil {
			log.Println("err", err)
		}
	} else {
		// 要不要更新截图？ 删除旧图 todo

		// 来源是插件，保留之前的tag
		if u.From == "ext" {
			oldTas := internal.NewTagInternal().GetTags(bookmark.Tags)
			for _, ta := range oldTas {
				tags = append(tags, ta.Name)
			}
		}
	}

	book := bookmarkInternal.CreateOrEdit(bookmark, tags)

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

func (that Bookmarks) deleteByUrl(w http.ResponseWriter, r *http.Request) {
	// 校验参数
	valid := validation.Validation{}
	body := apiV2.NewBody(r)
	u := struct {
		Url string `valid:"Required;"`
		Uid int
	}{
		Url: body.PostString("url"),
		Uid: int(apiV2.GetLoginUid(r)),
	}
	if err := valid.Submit(u); err != nil {
		api.Error(w, r, nil, err.Error(), -1)
		return
	}

	bookmarkInternal := internal.NewBookmarksInternal()
	bookmark := bookmarkInternal.Info(u.Url)

	if bookmark.ID <= 0 {
		apiV2.Error(w, r, errors.New("书签不存在"))
		return
	} else {
		bookmarkInternal.Delete(bookmark)
	}

	apiV2.Success(w, r, nil, "删除成功")
	return
}

func (that Bookmarks) showShot(w http.ResponseWriter, r *http.Request) {
	image_url := apiV2.QueryString(r, "image_url")

	log.Println("x", image_url)

	// 拼接头像文件路径
	filePath := filepath.Join(consts.UploadDir, image_url)

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Avatar not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// 设置HTTP头
	ext := filepath.Ext(filePath)
	switch ext {
	case ".jpg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	default:
		w.Header().Set("Content-Type", "image/jpeg")
	}

	w.Header().Set("Content-Disposition", "inline")
	w.Header().Del("Content-Length")

	// 将文件复制到response
	io.Copy(w, file)
}

func (that Bookmarks) saveImageFromBase64(base64Str string) (string, error) {
	if len(base64Str) == 0 {
		return "", apiV2.NewApiError(int(errorcode.ErrorBookmarkBase64Empty), errorcode.ErrorBookmarkBase64Empty.String())
	}
	// 1. 从base64中解析出mime类型
	index := strings.Index(base64Str, ",")
	base64Data := base64Str[index+1:]

	// 4. 解码base64字符串获取图片数据
	imgData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", apiV2.NewApiError(int(errorcode.ErrorBookmarkBase64Error), errorcode.ErrorBookmarkBase64Error.String()+" > "+err.Error())
	}

	// 5. 构造文件名及文件路径
	fileName := fmt.Sprintf("%d.jpg", time.Now().UnixNano())
	filePath := filepath.Join(consts.UploadDir, fileName)

	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}

	// 6. 将图片数据写入文件
	out, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		return "", apiV2.NewApiError(int(errorcode.ErrorBookmarkBase64WriteFile), errorcode.ErrorBookmarkBase64WriteFile.String()+" > "+err.Error())
	}
	defer out.Close()

	// 7. 编码图片信息
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return "", apiV2.NewApiError(int(errorcode.ErrorBookmarkBase64Decode), errorcode.ErrorBookmarkBase64Decode.String()+" > "+err.Error())
	}
	jpeg.Encode(out, img, &jpeg.Options{Quality: 95})

	return fileName, nil
}
