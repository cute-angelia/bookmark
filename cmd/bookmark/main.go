package main

import (
	accountsCtl "bookmark/cmd/bookmark/controller/accounts"
	"bookmark/cmd/bookmark/controller/auth"
	"bookmark/cmd/bookmark/controller/bookmarks"
	"bookmark/cmd/bookmark/controller/tags"
	"bookmark/cmd/bookmark/database"
	"bookmark/cmd/bookmark/model"
	"bookmark/pkg/configV2"
	"bookmark/pkg/env"
	"bookmark/pkg/imiddleware"
	"context"
	"embed"
	"fmt"
	"github.com/cute-angelia/go-utils/components/caches/ibunt"
	"github.com/cute-angelia/go-utils/components/igorm"
	"github.com/cute-angelia/go-utils/components/loggers/loggerV3"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

const PortSite = ":38112"
const ProjectName = "bookmark-api"

var BuildDbEnv = "" // db 连接环境

//go:embed view/*
var Assets embed.FS

func main() {
	// init logger
	loggerV3.New(loggerV3.WithIsOnline(!env.IsLocal()), loggerV3.WithFileJson(false))

	// 初始化配置
	configV2.InitConfig("")

	// 初始化 buntcache
	ibunt.New()

	if env.IsLocal() {
	}

	// init shiori
	dataDir := os.Getenv("SHIORI_DIR")
	dbPath := filepath.Join(dataDir, "bookmark.db")

	database.Dbx, _ = database.ConnectSqlite(dbPath)
	if err := database.Dbx.Migrate(); err != nil {
		log.Println("db.Migrate", err)
	}

	accounts, err := database.Dbx.GetAccounts(context.Background(), database.GetAccountsOptions{})
	if err != nil {
		log.Printf("Failed to get owner account: %v\n", err)
		os.Exit(1)
	}

	if len(accounts) == 0 {
		account := model.AccountModel{
			Username: "admin",
			Password: "admin123",
			Owner:    true,
		}

		if err := database.Dbx.SaveAccount(context.Background(), account); err != nil {
			log.Println("error ensuring owner account")
		}
	}

	// init
	igorm.New(
		igorm.WithDbName("cache"),
		igorm.WithDbFile(dbPath),
		igorm.WithDebug(true),
		igorm.WithLoggerWriter(loggerV3.GetLogger()),
	).MustInitSqlite()

	// 定义路由
	r := chi.NewRouter()
	corsz := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"hi,world"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	r.Use(corsz.Handler)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	// 开启日志
	//r.Use(middleware.Logger)
	// 初始化

	r.Use(imiddleware.Jwt([]string{
		"/api/auth/login",
		"/favicon.ico",
		"/",
		"/login.html",
		"/assets/.*",
		"/api/bookmarks/showShot",
	}), imiddleware.Log([]string{}))

	// 路由组
	r.Group(func(r chi.Router) {
		r.Route("/api", func(r chi.Router) {
			// 登录相关
			r.Mount("/auth", auth.Auth{}.Routes())

			r.Mount("/bookmarks", bookmarks.Bookmarks{}.Routes())
			r.Mount("/tags", tags.Tags{}.Routes())
			r.Mount("/accounts", accountsCtl.Accounts{}.Routes())

			//// 账号
			//r.Mount("/account", user.Account{}.Routes())
			//
			//// 资源
			//r.Mount("/item", user.Item{}.Routes())
			//
			//// building
			//r.Mount("/building", user.Building{}.Routes())
			//
			//// hero
			//r.Mount("/hero", user.Hero{}.Routes())
			//
			//// team
			//r.Mount("/team", user.Team{}.Routes())
			//
			//// props
			//r.Mount("/props", user.Props{}.Routes())
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		err := tryRead(Assets, "view", r.URL.Path, w)
		if err == nil {
			return
		}
		err = tryRead(Assets, "view", "index.html", w)
		if err != nil {
			log.Println(err)
		}
	})

	log.Println(fmt.Sprintf("启动成功~  %s http://127.0.0.1%s", ProjectName, PortSite))
	// dev debug

	if err := http.ListenAndServe(PortSite, r); err != nil {
		log.Println(err)
	}
}

func tryRead(fs embed.FS, prefix, requestedPath string, w http.ResponseWriter) error {
	f, err := fs.Open(path.Join(prefix, requestedPath))
	if err != nil {
		return err
	}
	defer f.Close()

	// Goのfs.Openはディレクトリを読みこもとうしてもエラーにはならないがここでは邪魔なのでエラー扱いにする
	stat, _ := f.Stat()
	if stat.IsDir() {
		return errors.New("path is dir")
	}
	contentType := mime.TypeByExtension(filepath.Ext(requestedPath))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "max-age=864000")
	_, err = io.Copy(w, f)
	return err
}
