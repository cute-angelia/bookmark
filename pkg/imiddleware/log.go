/*
*
日志中间件
计算请求耗时
配合 jwt 中间件使用（获取, jwt_uid， jwt_cid）
*/
package imiddleware

import (
	"bytes"
	"fmt"
	"github.com/cute-angelia/go-utils/utils/http/apiV2"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ===== 请勿删除， 可支持传入配置 =====
var noCacheLogPaths = []string{}

func Log(allowList []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			if apiV2.GetContentType(r.Header.Get("Content-Type")) == apiV2.ContentTypeJSON {
				body, _ := io.ReadAll(r.Body)
				log.Printf(" 请求地址: %s,  请求数据: %s,", r.URL, string(body))
				r.Body = io.NopCloser(bytes.NewBuffer(body))
			}

			// ===== 使用传入配置 =====
			if len(allowList) > 0 {
				noCacheLogPaths = allowList
			}

			// 例外直接过
			for _, v := range noCacheLogPaths {
				// 正则匹配
				if strings.Contains(v, "*") {
					r1 := regexp.MustCompile(v)
					if r1.MatchString(r.URL.Path) {
						next.ServeHTTP(w, r)
						return
					}
				}

				// 路径匹配
				if r.URL.Path == v {
					// 不登录也需要设置头部
					next.ServeHTTP(w, r)
					return
				}
			}

			// 逻辑处理
			// jwt_uid := r.Header.Get("jwt_uid")
			r.Header.Set("jwt_app_start_time", fmt.Sprintf("%d", time.Now().Unix()))

			// 逻辑处理 end

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
