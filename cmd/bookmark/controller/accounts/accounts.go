package accounts

import (
	"bookmark/cmd/bookmark/database"
	"bookmark/cmd/bookmark/model"
	"context"
	"github.com/cute-angelia/go-utils/components/igorm"
	"github.com/cute-angelia/go-utils/utils/http/api"
	"github.com/cute-angelia/go-utils/utils/http/apiV2"
	"github.com/cute-angelia/go-utils/utils/http/validation"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
)

type Accounts struct {
}

func (that Accounts) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", that.list)
	r.Post("/add", that.add)
	r.Post("/delete", that.delete)

	r.Post("/changePwd", that.changePwd)
	return r
}

func (that Accounts) list(w http.ResponseWriter, r *http.Request) {
	// 校验参数
	valid := validation.Validation{}
	u := struct {
		Uid int32 `valid:"Required;"`
	}{
		Uid: apiV2.GetLoginUid(r),
	}
	if err := valid.Submit(u); err != nil {
		api.Error(w, r, nil, err.Error(), -1)
		return
	}

	account, _ := database.Dbx.GetAccounts(context.Background(), database.GetAccountsOptions{})

	api.Success(w, r, account, "登录成功")
	return
}

func (that Accounts) add(w http.ResponseWriter, r *http.Request) {
	// 校验参数
	body := apiV2.NewBody(r)
	valid := validation.Validation{}
	u := struct {
		Uid      int32  `valid:"Required;"`
		Username string `valid:"Required;"`
		Password string `valid:"Required;"`
		Owner    bool
	}{
		Uid:      apiV2.GetLoginUid(r),
		Username: body.PostString("username"),
		Password: body.PostString("password"),
		Owner:    body.PostBool("owner"),
	}
	if err := valid.Submit(u); err != nil {
		apiV2.Error(w, r, err)
		return
	}

	log.Print("u.Owner", u.Owner)

	orm, _ := igorm.GetGormSQLite("cache")
	account := model.AccountModel{}
	orm.Where("username = ?", u.Username).First(&account)

	if account.ID > 0 {
		apiV2.Error(w, r, errors.New("账号已存在"))
		return
	} else {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(u.Password), 10)
		account.Username = u.Username
		account.Password = string(hashedPassword)
		account.Owner = u.Owner
		orm.Create(&account)
	}

	apiV2.Success(w, r, account, "添加成功")
	return
}

func (that Accounts) delete(w http.ResponseWriter, r *http.Request) {
	// 校验参数
	body := apiV2.NewBody(r)
	valid := validation.Validation{}
	u := struct {
		Uid      int32  `valid:"Required;"`
		Username string `valid:"Required;"`
	}{
		Uid:      apiV2.GetLoginUid(r),
		Username: body.PostString("username"),
	}
	if err := valid.Submit(u); err != nil {
		apiV2.Error(w, r, err)
		return
	}

	orm, _ := igorm.GetGormSQLite("cache")

	adminUser := model.AccountModel{}
	orm.Where("id = ?", u.Uid).First(&adminUser)
	if !adminUser.Owner {
		apiV2.Error(w, r, errors.New("非管理员不能删除账号"))
		return
	}

	account := model.AccountModel{}
	orm.Where("username = ?", u.Username).First(&account)

	if account.ID <= 0 {
		apiV2.Error(w, r, errors.New("账号不存在"))
		return
	} else {
		orm.Table(account.TableName()).Where("username = ?", u.Username).Delete(&account)
	}

	apiV2.Success(w, r, account, "删除成功")
	return
}

func (that Accounts) changePwd(w http.ResponseWriter, r *http.Request) {
	// 校验参数
	body := apiV2.NewBody(r)
	valid := validation.Validation{}
	u := struct {
		Uid         int32  `valid:"Required;"`
		Username    string `valid:"Required;"`
		Password    string `valid:"Required;"`
		OldPassword string `valid:"Required;"`
		Owner       bool
	}{
		Uid:         apiV2.GetLoginUid(r),
		Username:    body.PostString("username"),
		Password:    body.PostString("password"),
		OldPassword: body.PostString("oldPassword"),
		Owner:       body.PostBool("owner"),
	}
	if err := valid.Submit(u); err != nil {
		apiV2.Error(w, r, err)
		return
	}

	orm, _ := igorm.GetGormSQLite("cache")

	account := model.AccountModel{}
	orm.Where("username = ?", u.Username).First(&account)

	if account.ID <= 0 {
		apiV2.Error(w, r, errors.New("账号不存在"))
		return
	}

	loginUser := model.AccountModel{}
	orm.Where("id = ?", u.Uid).First(&loginUser)
	if account.ID != loginUser.ID && !loginUser.Owner {
		apiV2.Error(w, r, errors.New("非管理员不能修改他人账号"))
		return
	}

	// 校验旧密码
	err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(u.OldPassword))
	if err != nil {
		api.Error(w, r, nil, "密码不匹配", -1)
		return
	} else {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(u.Password), 10)
		account.Password = string(hashedPassword)
		account.Owner = u.Owner
		orm.Save(&account)
		apiV2.Success(w, r, account, "修改成功")
		return
	}
}
