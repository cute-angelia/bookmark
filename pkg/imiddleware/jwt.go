package imiddleware

import (
	"fmt"
	"github.com/cute-angelia/go-utils/components/caches/ibunt"
	"github.com/cute-angelia/go-utils/utils/http/api"
	"github.com/cute-angelia/go-utils/utils/http/jwt"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// ===== 请勿删除， 可支持传入配置 =====
var allowLoginPaths = []string{}

const JWTSECTET = "https://github.com/cute-angelia/shiori"

/*
*
设置头部信息， 包括未登陆的
*/
func setHeaderInfo(r *http.Request, jwtToken string) {

	// 设置 cid
	cid := r.URL.Query().Get("cid")
	r.Header.Set("jwt_cid", fmt.Sprintf("%v", cid))

	appid := r.URL.Query().Get("appid")
	r.Header.Set("jwt_appid", fmt.Sprintf("%v", appid))

	if len(jwtToken) > 0 {
		jwttobj := jwt.NewJwt(JWTSECTET)
		if z, err := jwttobj.Decode(jwtToken); err != nil {
			return
		} else {
			// log.Println("jwttobj", z)
			// 只认这个 uid
			uid, _ := z.Get("uid")
			openid, _ := z.Get("openid")
			cid, _ := z.Get("cid")
			nickname, _ := z.Get("nickname")
			username, _ := z.Get("username")
			avatar, _ := z.Get("avatar")
			appid, _ := z.Get("appid")
			froms, _ := z.Get("froms")

			if uid != nil {
				r.Header.Set("jwt_uid", fmt.Sprintf("%v", uid))
			}

			if froms != nil {
				r.Header.Set("jwt_froms", fmt.Sprintf("%v", froms))
			}

			if appid != nil && fmt.Sprintf("%v", appid) != "" {
				r.Header.Set("jwt_appid", fmt.Sprintf("%v", appid))
			}

			if openid != nil {
				r.Header.Set("jwt_openid", fmt.Sprintf("%v", openid))
			}

			icid := fmt.Sprintf("%v", cid)
			if cid != nil && icid != "0" {
				r.Header.Set("jwt_cid", fmt.Sprintf("%v", cid))
			}

			if nickname != nil {
				r.Header.Set("jwt_nickname", fmt.Sprintf("%v", nickname))
			}
			if username != nil {
				r.Header.Set("jwt_username", fmt.Sprintf("%v", username))
			}
			if avatar != nil {
				r.Header.Set("jwt_avatar", fmt.Sprintf("%v", avatar))
			}
			r.Header.Set("jwt_token", jwtToken)
		}
	}
}

func Jwt(allowList []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			// 提取 Token
			authToken := r.Header.Get("Authorization")
			jwtToken := ""

			if len(authToken) > 0 && strings.Contains(authToken, "Bearer") {
				allToken := strings.Split(authToken, "Bearer ")
				if len(allToken) == 2 {
					jwtToken = allToken[1]
				}
			}
			// ===== 使用传入配置 =====
			if len(allowList) > 0 {
				allowLoginPaths = allowList
			}

			// 例外直接过
			for _, v := range allowLoginPaths {
				// 正则匹配
				if strings.Contains(v, "*") {
					r1 := regexp.MustCompile(v)
					if r1.MatchString(r.URL.Path) {
						// log.Println("例外匹配过滤JWT √：", r1.MatchString(r.URL.Path), v, r.URL.Path)
						setHeaderInfo(r, jwtToken)
						next.ServeHTTP(w, r)
						return
					}
				}

				// 路径匹配
				if r.URL.Path == v {
					// 不登录也需要设置头部
					// log.Println("例外匹配过滤JWT √：", r.URL.Path)
					setHeaderInfo(r, jwtToken)
					next.ServeHTTP(w, r)
					return
				}
			}

			// 获取授权 header
			if len(jwtToken) < 30 {
				log.Println("例外匹配过滤JWT ×：", -999, r.URL.Path, jwtToken)
				api.Error(w, r, nil, "登录已过期, 请重新登录", -999)
				return
			}

			// check logout
			isLogOut := ibunt.Get("cache", jwtToken)
			if isLogOut == "true" {
				log.Println("例外匹配过滤JWT ×：", -999, r.URL.Path, jwtToken)
				api.Error(w, r, nil, "登录已过期, 请重新登录", -999)
				return
			}

			// jwt
			jwttobj := jwt.NewJwt(JWTSECTET)
			if _, err := jwttobj.Decode(jwtToken); err != nil {
				w.WriteHeader(203)
				log.Println("例外匹配过滤JWT ×：", -999, r.URL.Path, jwtToken)
				api.Error(w, r, nil, "登录已过期, 请重新登录", -999)
				return
			} else {
				setHeaderInfo(r, jwtToken)
			}
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func JwtNoNeedLogin(allowList []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			// 提取 Token
			authToken := r.Header.Get("Authorization")
			jwtToken := ""

			if len(authToken) > 0 && strings.Contains(authToken, "Bearer") {
				allToken := strings.Split(authToken, "Bearer ")
				if len(allToken) == 2 {
					jwtToken = allToken[1]
				}
			}
			setHeaderInfo(r, jwtToken)
			next.ServeHTTP(w, r)
			return
		}
		return http.HandlerFunc(fn)
	}
}
