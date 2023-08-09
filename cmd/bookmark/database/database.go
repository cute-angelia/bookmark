package database

import (
	model2 "bookmark/cmd/bookmark/model"
	"context"
	"embed"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

//go:embed migrations/*
var migrations embed.FS

// OrderMethod is the order method for getting bookmarks
type OrderMethod int

const (
	// DefaultOrder is oldest to newest.
	DefaultOrder OrderMethod = iota
	// ByLastAdded is from newest addition to the oldest.
	ByLastAdded
	// ByLastModified is from latest modified to the oldest.
	ByLastModified
)

// GetBookmarksOptions is options for fetching bookmarks from database.
type GetBookmarksOptions struct {
	IDs          []int
	Tags         []string
	ExcludedTags []string
	Keyword      string
	WithContent  bool
	OrderMethod  OrderMethod
	Limit        int
	Offset       int
}

// GetAccountsOptions is options for fetching accounts from database.
type GetAccountsOptions struct {
	Keyword string
	Owner   bool
}

func ConnectSqlite(dbPath string) (DB, error) {
	return OpenSQLiteDatabase(context.Background(), dbPath)
}

var Dbx DB

// DB is interface for accessing and manipulating data in database.
type DB interface {
	// Migrate runs migrations for this database
	Migrate() error

	// SaveBookmarks saves bookmarks data to database.
	SaveBookmarks(ctx context.Context, create bool, bookmarks ...model2.BookmarkModel) ([]model2.BookmarkModel, error)

	// GetBookmarks fetch list of bookmarks based on submitted options.
	GetBookmarks(ctx context.Context, opts GetBookmarksOptions) ([]model2.BookmarkModel, error)

	// GetBookmarksCount get count of bookmarks in database.
	GetBookmarksCount(ctx context.Context, opts GetBookmarksOptions) (int, error)

	// DeleteBookmarks removes all record with matching ids from database.
	DeleteBookmarks(ctx context.Context, ids ...int) error

	// GetBookmark fetchs bookmark based on its ID or URL.
	GetBookmark(ctx context.Context, id int, url string) (model2.BookmarkModel, bool, error)

	// SaveAccount saves new account in database
	SaveAccount(ctx context.Context, a model2.AccountModel) error

	// GetAccounts fetch list of account (without its password) with matching keyword.
	GetAccounts(ctx context.Context, opts GetAccountsOptions) ([]model2.AccountModel, error)

	// GetAccount fetch account with matching username.
	GetAccount(ctx context.Context, username string) (model2.AccountModel, bool, error)

	// DeleteAccounts removes all record with matching usernames
	DeleteAccounts(ctx context.Context, usernames ...string) error

	// CreateTags creates new tags in database.
	CreateTags(ctx context.Context, tags ...model2.TagModel) error

	// GetTags fetch list of tags and its frequency from database.
	GetTags(ctx context.Context) ([]model2.TagModel, error)

	// RenameTag change the name of a tag.
	RenameTag(ctx context.Context, id int, newName string) error
}

type dbbase struct {
	sqlx.DB
}

func (db *dbbase) withTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	defer func() {
		if err := tx.Commit(); err != nil {
			log.Printf("error during commit: %s", err)
		}
	}()

	err = fn(tx)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			log.Printf("error during rollback: %s", err)
		}
		return errors.WithStack(err)
	}

	return err
}
