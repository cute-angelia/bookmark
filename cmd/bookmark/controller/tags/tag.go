package tags

import (
	"bookmark/cmd/bookmark/internal"
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
		Keyword string
	}{
		Keyword: body.PostString("keyword"),
	}
	if err := valid.Submit(u); err != nil {
		api.Error(w, r, nil, err.Error(), -1)
		return
	}

	api.Success(w, r, internal.NewTagInternal().GetList(), "获取书签列表")
	return
}
