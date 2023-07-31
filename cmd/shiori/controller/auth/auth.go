package auth

import (
	"bookmark/cmd/shiori/model"
	"github.com/cute-angelia/go-utils/components/igorm"
	"github.com/cute-angelia/go-utils/utils/http/api"
	"github.com/cute-angelia/go-utils/utils/http/apiV2"
	"github.com/cute-angelia/go-utils/utils/http/jwt"
	"github.com/cute-angelia/go-utils/utils/http/validation"
	"github.com/go-chi/chi"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

// Auth 登录
type Auth struct {
}

func (that Auth) Routes() chi.Router {
	r := chi.NewRouter()
	// 登录
	r.Post("/login", that.login)
	r.Post("/logout", that.logout)
	return r
}

func (that Auth) login(w http.ResponseWriter, r *http.Request) {
	// 校验参数
	body := apiV2.NewBody(r)
	valid := validation.Validation{}
	u := struct {
		Username string `valid:"Required;"`
		Password string `valid:"Required;"`
	}{
		Username: body.PostString("username"),
		Password: body.PostString("password"),
	}
	if err := valid.Submit(u); err != nil {
		api.Error(w, r, nil, err.Error(), -1)
		return
	}

	orm, _ := igorm.GetGormSQLite("cache")
	account := model.AccountModel{}
	orm.Where("username = ?", u.Username).First(&account)

	// 校验密码
	err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(u.Password))
	if err != nil {
		api.Error(w, r, nil, "密码不匹配", -1)
		return
	}

	resp := struct {
		Token string `json:"token"`
	}{
		Token: that.jwtToken(account),
	}

	api.Success(w, r, resp, "登录成功")
	return
}

func (that Auth) logout(w http.ResponseWriter, r *http.Request) {
	return
}

func (that Auth) jwtToken(account model.AccountModel) string {
	exp := time.Now().AddDate(10, 6, 9).Unix()
	jwttobj := jwt.NewJwt("https://github.com/cute-angelia/shiori")
	jwtToken, _ := jwttobj.Generate(map[string]interface{}{
		"uid":      account.ID,
		"username": account.Username,
		"owner":    account.Owner,
		"exp":      exp,
	})
	return jwtToken
}
