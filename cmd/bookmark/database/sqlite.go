package database

import (
	model2 "bookmark/cmd/bookmark/model"
	"context"
	"database/sql"
	"github.com/cute-angelia/go-utils/syntax/itime"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"strings"
)

// SQLiteDatabase is implementation of Database interface
// for connecting to SQLite3 database.
type SQLiteDatabase struct {
	dbbase
}

type bookmarkContent struct {
	ID      int    `db:"docid"`
	Content string `db:"content"`
	HTML    string `db:"html"`
}

type tagContent struct {
	ID int `db:"bookmark_id"`
	model2.TagModel
}

// OpenSQLiteDatabase creates and open connection to new SQLite3 database.
func OpenSQLiteDatabase(ctx context.Context, databasePath string) (sqliteDB *SQLiteDatabase, err error) {
	// Open database
	db, err := sqlx.ConnectContext(ctx, "sqlite", databasePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sqliteDB = &SQLiteDatabase{dbbase: dbbase{*db}}
	return sqliteDB, nil
}

// Migrate runs migrations for this database engine
func (db *SQLiteDatabase) Migrate() error {
	sourceDriver, err := iofs.New(migrations, "migrations/sqlite")
	if err != nil {
		return errors.WithStack(err)
	}

	dbDriver, err := sqlite.WithInstance(db.DB.DB, &sqlite.Config{})
	if err != nil {
		return errors.WithStack(err)
	}

	migration, err := migrate.NewWithInstance(
		"iofs",
		sourceDriver,
		"sqlite",
		dbDriver,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

// SaveBookmarks saves new or updated bookmarks to database.
// Returns the saved ID and error message if any happened.
func (db *SQLiteDatabase) SaveBookmarks(ctx context.Context, create bool, bookmarks ...model2.BookmarkModel) ([]model2.BookmarkModel, error) {
	var result []model2.BookmarkModel

	if err := db.withTx(ctx, func(tx *sqlx.Tx) error {
		// Prepare statement

		stmtInsertBook, err := tx.PreparexContext(ctx, `INSERT INTO bookmark
			(url, title, excerpt, uid, public, modified )
			VALUES(?, ?, ?, ?, ?, ?, ?) RETURNING id`)
		if err != nil {
			return errors.WithStack(err)
		}

		stmtUpdateBook, err := tx.PreparexContext(ctx, `UPDATE bookmark SET
			url = ?, title = ?,	excerpt = ?, uid = ?,
			public = ?, modified = ? 
			WHERE id = ?`)
		if err != nil {
			return errors.WithStack(err)
		}

		stmtInsertBookContent, err := tx.PreparexContext(ctx, `INSERT OR REPLACE INTO bookmark_content
			(docid, title, content, html)
			VALUES (?, ?, ?, ?)`)
		if err != nil {
			return errors.WithStack(err)
		}

		stmtUpdateBookContent, err := tx.PreparexContext(ctx, `UPDATE bookmark_content SET
			title = ?, content = ?, html = ?
			WHERE docid = ?`)
		if err != nil {
			return errors.WithStack(err)
		}

		stmtGetTag, err := tx.PreparexContext(ctx, `SELECT id FROM tag WHERE name = ?`)
		if err != nil {
			return errors.WithStack(err)
		}

		stmtInsertTag, err := tx.PreparexContext(ctx, `INSERT INTO tag (name) VALUES (?)`)
		if err != nil {
			return errors.WithStack(err)
		}

		stmtInsertBookTag, err := tx.PreparexContext(ctx, `INSERT OR IGNORE INTO bookmark_tag
			(tag_id, bookmark_id) VALUES (?, ?)`)
		if err != nil {
			return errors.WithStack(err)
		}
		if err != nil {
			return errors.WithStack(err)
		}

		// Prepare modified time
		modifiedTime := itime.NewUnixNow().Format()

		// Execute statements

		for _, book := range bookmarks {
			// Check URL and title
			if book.URL == "" {
				return errors.New("URL must not be empty")
			}

			if book.Title == "" {
				return errors.New("title must not be empty")
			}

			// Set modified time
			if book.Modified == "" {
				book.Modified = modifiedTime
			}

			// Create or update bookmark
			var err error
			if create {
				err = stmtInsertBook.QueryRowContext(ctx,
					book.URL, book.Title, book.Excerpt, book.Uid, book.Public, book.Modified).Scan(&book.ID)
			} else {
				_, err = stmtUpdateBook.ExecContext(ctx,
					book.URL, book.Title, book.Excerpt, book.Uid, book.Public, book.Modified, book.ID)
			}
			if err != nil {
				return errors.WithStack(err)
			}

			// Try to update it first to check for existence, we can't do an UPSERT here because
			// bookmant_content is a virtual table
			res, err := stmtUpdateBookContent.ExecContext(ctx, book.Title, book.ID)
			if err != nil {
				return errors.WithStack(err)
			}

			rows, err := res.RowsAffected()
			if err != nil {
				return errors.WithStack(err)
			}

			if rows == 0 {
				_, err = stmtInsertBookContent.ExecContext(ctx, book.ID, book.Title)
				if err != nil {
					return errors.WithStack(err)
				}
			}

			// Save book tags
			newTags := []model2.TagModel{}
			for _, tag := range book.TagsDetail {
				// If it's deleted tag, delete and continue
				// Normalize tag name
				tagName := strings.ToLower(tag.Name)
				tagName = strings.Join(strings.Fields(tagName), " ")

				// If tag doesn't have any ID, fetch it from database
				if tag.ID == 0 {
					if err := stmtGetTag.GetContext(ctx, &tag.ID, tagName); err != nil && err != sql.ErrNoRows {
						return errors.WithStack(err)
					}

					// If tag doesn't exist in database, save it
					if tag.ID == 0 {
						res, err := stmtInsertTag.ExecContext(ctx, tagName)
						if err != nil {
							return errors.WithStack(err)
						}

						tagID64, err := res.LastInsertId()
						if err != nil && err != sql.ErrNoRows {
							return errors.WithStack(err)
						}

						tag.ID = int(tagID64)
					}

					if _, err := stmtInsertBookTag.ExecContext(ctx, tag.ID, book.ID); err != nil {
						return errors.WithStack(err)
					}
				}

				newTags = append(newTags, tag)
			}

			book.TagsDetail = newTags
			result = append(result, book)
		}

		return nil
	}); err != nil {
		return nil, errors.WithStack(err)
	}

	return result, nil
}

// GetBookmarks fetch list of bookmarks based on submitted options.
func (db *SQLiteDatabase) GetBookmarks(ctx context.Context, opts GetBookmarksOptions) ([]model2.BookmarkModel, error) {
	// Create initial query
	query := `SELECT
		b.id,
		b.url,
		b.title,
		b.image_url,
		b.excerpt,
		b.uid,
		b.public,
		b.modified
		FROM bookmark b
		WHERE 1`

	// Add where clause
	args := []interface{}{}

	// Add where clause for IDs
	if len(opts.IDs) > 0 {
		query += ` AND b.id IN (?)`
		args = append(args, opts.IDs)
	}

	// Add where clause for search keyword
	if opts.Keyword != "" {
		query += ` AND (b.url LIKE ? OR b.excerpt LIKE ? )`

		args = append(args,
			"%"+opts.Keyword+"%",
			"%"+opts.Keyword+"%")
		// Replace dash with spaces since FTS5 uses `-name` as column identifier
		//opts.Keyword = strings.Replace(opts.Keyword, "-", " ", -1)
		//args = append(args, opts.Keyword, opts.Keyword)
	}

	// Add where clause for tags.
	// First we check for * in excluded and included tags,
	// which means all tags will be excluded and included, respectively.
	excludeAllTags := false
	for _, excludedTag := range opts.ExcludedTags {
		if excludedTag == "*" {
			excludeAllTags = true
			opts.ExcludedTags = []string{}
			break
		}
	}

	includeAllTags := false
	for _, includedTag := range opts.Tags {
		if includedTag == "*" {
			includeAllTags = true
			opts.Tags = []string{}
			break
		}
	}

	// If all tags excluded, we will only show bookmark without tags.
	// In other hand, if all tags included, we will only show bookmark with tags.
	if excludeAllTags {
		query += ` AND b.id NOT IN (SELECT DISTINCT bookmark_id FROM bookmark_tag)`
	} else if includeAllTags {
		query += ` AND b.id IN (SELECT DISTINCT bookmark_id FROM bookmark_tag)`
	}

	// Now we only need to find the normal tags
	if len(opts.Tags) > 0 {
		query += ` AND b.id IN (
			SELECT bt.bookmark_id
			FROM bookmark_tag bt
			LEFT JOIN tag t ON bt.tag_id = t.id
			WHERE t.name IN(?)
			GROUP BY bt.bookmark_id
			HAVING COUNT(bt.bookmark_id) = ?)`

		args = append(args, opts.Tags, len(opts.Tags))
	}

	if len(opts.ExcludedTags) > 0 {
		query += ` AND b.id NOT IN (
			SELECT DISTINCT bt.bookmark_id
			FROM bookmark_tag bt
			LEFT JOIN tag t ON bt.tag_id = t.id
			WHERE t.name IN(?))`

		args = append(args, opts.ExcludedTags)
	}

	// Add order clause
	switch opts.OrderMethod {
	case ByLastAdded:
		query += ` ORDER BY b.id DESC`
	case ByLastModified:
		query += ` ORDER BY b.modified DESC`
	default:
		query += ` ORDER BY b.id`
	}

	if opts.Limit > 0 && opts.Offset >= 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, opts.Limit, opts.Offset)
	}

	// Expand query, because some of the args might be an array
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Fetch bookmarks
	bookmarks := []model2.BookmarkModel{}
	err = db.SelectContext(ctx, &bookmarks, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.WithStack(err)
	}

	// store bookmark IDs for further enrichment
	var bookmarkIds = make([]int, 0, len(bookmarks))
	for _, book := range bookmarks {
		bookmarkIds = append(bookmarkIds, book.ID)
	}

	if len(bookmarkIds) == 0 {
		return bookmarks, nil
	}

	// Fetch tags for each bookmark
	tags := make([]tagContent, 0, len(bookmarks))
	tagsMap := make(map[int][]model2.TagModel, len(bookmarks))

	tagsQuery, tagArgs, err := sqlx.In(`SELECT bt.bookmark_id, t.id, t.name
		FROM bookmark_tag bt
		LEFT JOIN tag t ON bt.tag_id = t.id
		WHERE bt.bookmark_id IN (?)
		ORDER BY t.name`, bookmarkIds)
	tagsQuery = db.Rebind(tagsQuery)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = db.Select(&tags, tagsQuery, tagArgs...)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.WithStack(err)
	}
	for _, fetchedTag := range tags {
		if tags, found := tagsMap[fetchedTag.ID]; found {
			tagsMap[fetchedTag.ID] = append(tags, fetchedTag.TagModel)
		} else {
			tagsMap[fetchedTag.ID] = []model2.TagModel{fetchedTag.TagModel}
		}
	}
	for i := range bookmarks[:] {
		book := &bookmarks[i]
		if tags, found := tagsMap[book.ID]; found {
			book.TagsDetail = tags
		} else {
			book.TagsDetail = []model2.TagModel{}
		}
	}

	return bookmarks, nil
}

// GetBookmarksCount fetch count of bookmarks based on submitted options.
func (db *SQLiteDatabase) GetBookmarksCount(ctx context.Context, opts GetBookmarksOptions) (int, error) {
	// Create initial query
	query := `SELECT COUNT(b.id)
		FROM bookmark b
		WHERE 1`

	// Add where clause
	args := []interface{}{}

	// Add where clause for IDs
	if len(opts.IDs) > 0 {
		query += ` AND b.id IN (?)`
		args = append(args, opts.IDs)
	}

	// Add where clause for search keyword
	if opts.Keyword != "" {
		query += ` AND (b.url LIKE ? OR b.excerpt LIKE ?)`

		args = append(args,
			"%"+opts.Keyword+"%",
			"%"+opts.Keyword+"%",
		)
	}

	// Add where clause for tags.
	// First we check for * in excluded and included tags,
	// which means all tags will be excluded and included, respectively.
	excludeAllTags := false
	for _, excludedTag := range opts.ExcludedTags {
		if excludedTag == "*" {
			excludeAllTags = true
			opts.ExcludedTags = []string{}
			break
		}
	}

	includeAllTags := false
	for _, includedTag := range opts.Tags {
		if includedTag == "*" {
			includeAllTags = true
			opts.Tags = []string{}
			break
		}
	}

	// If all tags excluded, we will only show bookmark without tags.
	// In other hand, if all tags included, we will only show bookmark with tags.
	if excludeAllTags {
		query += ` AND b.id NOT IN (SELECT DISTINCT bookmark_id FROM bookmark_tag)`
	} else if includeAllTags {
		query += ` AND b.id IN (SELECT DISTINCT bookmark_id FROM bookmark_tag)`
	}

	// Now we only need to find the normal tags
	if len(opts.Tags) > 0 {
		query += ` AND b.id IN (
			SELECT bt.bookmark_id
			FROM bookmark_tag bt
			LEFT JOIN tag t ON bt.tag_id = t.id
			WHERE t.name IN(?)
			GROUP BY bt.bookmark_id
			HAVING COUNT(bt.bookmark_id) = ?)`

		args = append(args, opts.Tags, len(opts.Tags))
	}

	if len(opts.ExcludedTags) > 0 {
		query += ` AND b.id NOT IN (
			SELECT DISTINCT bt.bookmark_id
			FROM bookmark_tag bt
			LEFT JOIN tag t ON bt.tag_id = t.id
			WHERE t.name IN(?))`

		args = append(args, opts.ExcludedTags)
	}

	// Expand query, because some of the args might be an array
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	// Fetch count
	var nBookmarks int
	err = db.GetContext(ctx, &nBookmarks, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return 0, errors.WithStack(err)
	}

	return nBookmarks, nil
}

// DeleteBookmarks removes all record with matching ids from database.
func (db *SQLiteDatabase) DeleteBookmarks(ctx context.Context, ids ...int) error {
	if err := db.withTx(ctx, func(tx *sqlx.Tx) error {
		// Prepare queries
		delBookmark := `DELETE FROM bookmark`
		delBookmarkTag := `DELETE FROM bookmark_tag`
		delBookmarkContent := `DELETE FROM bookmark_content`

		// Delete bookmark(s)
		if len(ids) == 0 {
			_, err := tx.ExecContext(ctx, delBookmarkContent)
			if err != nil {
				return errors.WithStack(err)
			}

			_, err = tx.ExecContext(ctx, delBookmarkTag)
			if err != nil {
				return errors.WithStack(err)
			}

			_, err = tx.ExecContext(ctx, delBookmark)
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			delBookmark += ` WHERE id = ?`
			delBookmarkTag += ` WHERE bookmark_id = ?`
			delBookmarkContent += ` WHERE docid = ?`

			stmtDelBookmark, err := tx.Preparex(delBookmark)
			if err != nil {
				return errors.WithStack(err)
			}

			stmtDelBookmarkTag, err := tx.Preparex(delBookmarkTag)
			if err != nil {
				return errors.WithStack(err)
			}

			stmtDelBookmarkContent, err := tx.Preparex(delBookmarkContent)
			if err != nil {
				return errors.WithStack(err)
			}

			for _, id := range ids {
				_, err = stmtDelBookmarkContent.ExecContext(ctx, id)
				if err != nil {
					return errors.WithStack(err)
				}

				_, err = stmtDelBookmarkTag.ExecContext(ctx, id)
				if err != nil {
					return errors.WithStack(err)
				}

				_, err = stmtDelBookmark.ExecContext(ctx, id)
				if err != nil {
					return errors.WithStack(err)
				}
			}
		}

		return nil
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// GetBookmark fetches bookmark based on its ID or URL.
// Returns the bookmark and boolean whether it's exist or not.
func (db *SQLiteDatabase) GetBookmark(ctx context.Context, id int, url string) (model2.BookmarkModel, bool, error) {
	args := []interface{}{id}
	query := `SELECT
		b.id, b.url, b.title, b.excerpt, b.uid, b.public, b.modified,
		bc.content, bc.html 
		FROM bookmark b
		LEFT JOIN bookmark_content bc ON bc.docid = b.id
		WHERE b.id = ?`

	if url != "" {
		query += ` OR b.url = ?`
		args = append(args, url)
	}

	book := model2.BookmarkModel{}
	if err := db.GetContext(ctx, &book, query, args...); err != nil && err != sql.ErrNoRows {
		return book, false, errors.WithStack(err)
	}

	return book, book.ID != 0, nil
}

// SaveAccount saves new account to database. Returns error if any happened.
func (db *SQLiteDatabase) SaveAccount(ctx context.Context, account model2.AccountModel) error {
	if err := db.withTx(ctx, func(tx *sqlx.Tx) error {
		// Hash password with bcrypt
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(account.Password), 10)
		if err != nil {
			return err
		}

		// Insert account to database
		_, err = tx.Exec(`INSERT INTO account
		(username, password, owner) VALUES (?, ?, ?)
		ON CONFLICT(username) DO UPDATE SET
		password = ?, owner = ?`,
			account.Username, hashedPassword, account.Owner,
			hashedPassword, account.Owner)
		return errors.WithStack(err)
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// GetAccounts fetch list of account (without its password) based on submitted options.
func (db *SQLiteDatabase) GetAccounts(ctx context.Context, opts GetAccountsOptions) ([]model2.AccountModel, error) {
	// Create query
	args := []interface{}{}
	query := `SELECT id, username, owner FROM account WHERE 1`

	if opts.Keyword != "" {
		query += " AND username LIKE ?"
		args = append(args, "%"+opts.Keyword+"%")
	}

	if opts.Owner {
		query += " AND owner = 1"
	}

	query += ` ORDER BY username`

	// Fetch list account
	accounts := []model2.AccountModel{}
	err := db.SelectContext(ctx, &accounts, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.WithStack(err)
	}

	return accounts, nil
}

// GetAccount fetch account with matching username.
// Returns the account and boolean whether it's exist or not.
func (db *SQLiteDatabase) GetAccount(ctx context.Context, username string) (model2.AccountModel, bool, error) {
	account := model2.AccountModel{}
	if err := db.GetContext(ctx, &account, `SELECT
		id, username, password, owner FROM account WHERE username = ?`,
		username,
	); err != nil {
		return account, false, errors.WithStack(err)
	}

	return account, account.ID != 0, nil
}

// DeleteAccounts removes all record with matching usernames.
func (db *SQLiteDatabase) DeleteAccounts(ctx context.Context, usernames ...string) error {
	if err := db.withTx(ctx, func(tx *sqlx.Tx) error {
		// Delete account
		stmtDelete, err := tx.Preparex(`DELETE FROM account WHERE username = ?`)
		if err != nil {
			return errors.WithStack(err)
		}

		for _, username := range usernames {
			_, err := stmtDelete.ExecContext(ctx, username)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		return nil
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// CreateTags creates new tags from submitted objects.
func (db *SQLiteDatabase) CreateTags(ctx context.Context, tags ...model2.TagModel) error {
	query := `INSERT INTO tag (name) VALUES `
	values := []interface{}{}

	for _, t := range tags {
		query += "(?),"
		values = append(values, t.Name)
	}
	query = query[0 : len(query)-1]

	if err := db.withTx(ctx, func(tx *sqlx.Tx) error {
		stmt, err := tx.Preparex(query)
		if err != nil {
			return errors.Wrap(errors.WithStack(err), "error preparing query")
		}

		_, err = stmt.ExecContext(ctx, values...)
		if err != nil {
			return errors.Wrap(errors.WithStack(err), "error executing query")
		}

		return nil
	}); err != nil {
		return errors.Wrap(errors.WithStack(err), "error running transaction")
	}

	return nil
}

// GetTags fetch list of tags and their frequency.
func (db *SQLiteDatabase) GetTags(ctx context.Context) ([]model2.TagModel, error) {
	tags := []model2.TagModel{}
	query := `SELECT bt.tag_id id, t.name 
		FROM bookmark_tag bt
		LEFT JOIN tag t ON bt.tag_id = t.id
		GROUP BY bt.tag_id ORDER BY t.name`

	err := db.SelectContext(ctx, &tags, query)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.WithStack(err)
	}

	return tags, nil
}

// RenameTag change the name of a tag.
func (db *SQLiteDatabase) RenameTag(ctx context.Context, id int, newName string) error {
	if err := db.withTx(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE tag SET name = ? WHERE id = ?`, newName, id)
		return err
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
