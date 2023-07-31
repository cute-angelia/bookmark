package imiddleware

import (
	"fmt"
	"github.com/cute-angelia/go-utils/components/caches/ibunt"
	"github.com/cute-angelia/go-utils/utils/generator/hash"
	"github.com/cute-angelia/go-utils/utils/http/api"
	"log"
	"net/http"
	"time"
)

func Risk(inRankPaths []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// 例外直接过
			isMatch := false
			for _, v := range inRankPaths {
				// 路径匹配
				if r.URL.Path == v {
					isMatch = true
				}
			}

			// 匹配过滤路径直接过
			if isMatch {
				next.ServeHTTP(w, r)
				return
			} else {
				// 同一个路径 2 秒访问一次
				uid := r.Header.Get("jwt_uid")
				if len(uid) == 0 {
					next.ServeHTTP(w, r)
					return
				}
				zkey := fmt.Sprintf("middler_locked_%s_%s", uid, r.URL.Path)
				zkeymd5 := hash.NewEncodeMD5(zkey)
				if ok, _ := ibunt.IsNotLockedInLimit("cache", zkeymd5, time.Millisecond*150, ibunt.NewLockerOpt(ibunt.WithToday(false))); !ok {
					log.Println("risk locked:", zkey)
					api.Error(w, r, nil, "操作过快", -1000)
					return
				} else {
					next.ServeHTTP(w, r)
					return
				}
			}
			next.ServeHTTP(w, r)
			return
		}
		return http.HandlerFunc(fn)
	}
}
