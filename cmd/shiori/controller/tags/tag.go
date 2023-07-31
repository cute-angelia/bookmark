package tags

import (
	"bookmark/cmd/shiori/database"
	"context"
	"github.com/cute-angelia/go-utils/utils/http/api"
	"github.com/cute-angelia/go-utils/utils/http/apiV2"
	"github.com/cute-angelia/go-utils/utils/http/validation"
	"github.com/go-chi/chi"
	"net/http"
)

type Tags struct {
}

func (that Tags) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", that.lists)
	return r
}

func (that Tags) lists(w http.ResponseWriter, r *http.Request) {
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
	// Calculate max page
	if tags, err := database.Dbx.GetTags(context.Background()); err != nil {
		api.Error(w, r, nil, err.Error(), -1)
		return
	} else {
		api.Success(w, r, tags, "获取书签列表")
		return
	}
}
