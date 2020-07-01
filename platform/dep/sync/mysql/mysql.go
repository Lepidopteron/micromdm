package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	sq "gopkg.in/Masterminds/squirrel.v1"

	"github.com/micromdm/micromdm/platform/dep/sync"
)

type Mysql struct{ db *sqlx.DB }

func NewDB(db *sqlx.DB) (*Mysql, error) {
	// Required for TIMESTAMP DEFAULT 0
	_, err := db.Exec(`SET sql_mode = '';`)
	if err != nil {
		return nil, errors.Wrap(err, "setting sql_mode")
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS cursors (
		    value VARCHAR(128) PRIMARY KEY,
		    created_at TIMESTAMP DEFAULT 0
		);`)
	if err != nil {
		return nil, errors.Wrap(err, "creating cursors sql table failed")
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS dep_auto_assign (
		    profile_uuid VARCHAR(128) PRIMARY KEY,
		    filter TEXT DEFAULT NULL
		);`)
	if err != nil {
		return nil, errors.Wrap(err, "creating cursors sql table failed")
	}

	return &Mysql{db: db}, nil
}

func cursorColumns() []string {
	return []string{
		"value",
		"created_at",
	}
}

func depAutoAssignColumns() []string {
	return []string{
		"profile_uuid",
		"filter",
	}
}

const tableNameCursors = "cursors"
const tableNameDepAutoAssign = "dep_auto_assign"

func (d *Mysql) LoadCursor(ctx context.Context) (*sync.Cursor, error) {
	query, args, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Question).
		Select(cursorColumns()...).
		From(tableNameCursors).
		Where(sq.Eq{"value": "(SELECT MAX(created_at) FROM " + tableNameCursors + ")"}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "building sql")
	}

	var cursor = struct {
		Cursor sync.Cursor `json:"cursor"`
	}{}

	err = d.db.QueryRowxContext(ctx, query, args...).StructScan(&cursor.Cursor)
	if errors.Cause(err) == sql.ErrNoRows {
		return &sync.Cursor{}, nil
		return nil, cursorNotFoundErr{}
	}
	return &cursor.Cursor, errors.Wrap(err, "loading cursor")
}

func (d *Mysql) SaveCursor(ctx context.Context, cursor sync.Cursor) error {
	// Make sure we take the time offset into account for "zero" dates	
	t := time.Now()
	_, offset := t.Zone()

	// Don't multiply by zero
	if offset <= 0 {
		offset = 1
	}
	var minTimestampSec = int64(offset) * 60 * 60 * 24

	if cursor.CreatedAt.IsZero() || cursor.CreatedAt.Unix() < minTimestampSec {
		cursor.CreatedAt = time.Unix(minTimestampSec, 0)
	}

	updateQuery, argsUpdate, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Question).
		Update(tableNameCursors).
		Prefix("ON DUPLICATE KEY").
		Set("value", cursor.Value).
		Set("created_at", cursor.CreatedAt).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "building update query for cursor save")
	}

	// MySql Convention
	// Replace "ON DUPLICATE KEY UPDATE TABLE_NAME SET" to "ON DUPLICATE KEY UPDATE"
	updateQuery = strings.Replace(updateQuery, tableNameCursors+" SET ", "", -1)

	query, args, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Question).
		Insert(tableNameCursors).
		Columns(cursorColumns()...).
		Values(
			cursor.Value,
			cursor.CreatedAt,
		).
		Suffix(updateQuery).
		ToSql()

	var allArgs = append(args, argsUpdate...)

	if err != nil {
		return errors.Wrap(err, "building cursor save query")
	}

	_, err = d.db.ExecContext(ctx, query, allArgs...)

	return errors.Wrap(err, "exec cursor save in mysql")
}

func (d *Mysql) SaveAutoAssigner(ctx context.Context, a *sync.AutoAssigner) error {
	fmt.Println("SaveAutoAssigner")
	if a.Filter != "*" {
		return errors.New("only '*' filter auto-assigners supported")
	}
	updateQuery, argsUpdate, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Question).
		Update("dep_auto_assign").
		Prefix("ON DUPLICATE KEY").
		Set("profile_uuid", a.ProfileUUID).
		Set("filter", a.Filter).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "building update query for cursor save")
	}

	// MySql Convention
	// Replace "ON DUPLICATE KEY UPDATE TABLE_NAME SET" to "ON DUPLICATE KEY UPDATE"
	updateQuery = strings.Replace(updateQuery, "dep_auto_assign SET ", "", -1)

	query, args, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Question).
		Insert("dep_auto_assign").
		Columns("profile_uuid", "filter").
		Values(
			a.ProfileUUID,
			a.Filter,
		).
		Suffix(updateQuery).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "building cursor save query")
	}

	var allArgs = append(args, argsUpdate...)
	_, err = d.db.ExecContext(ctx, query, allArgs...)

	return errors.Wrap(err, "exec cursor save in mysql")
}

func (d *Mysql) DeleteAutoAssigner(ctx context.Context, filter string) error {
	query, args, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Question).
		Delete(tableNameDepAutoAssign).
		Where(sq.Eq{"filter": filter}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "building sql")
	}
	_, err = d.db.ExecContext(ctx, query, args...)
	return errors.Wrap(err, "delete autoassigner by filter")
}

func (d *Mysql) LoadAutoAssigners(ctx context.Context) ([]sync.AutoAssigner, error) {
	query, args, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Question).
		Select(depAutoAssignColumns()...).
		From(tableNameDepAutoAssign).
		ToSql()

	if err != nil {
		return nil, errors.Wrap(err, "building sql")
	}
	var list []sync.AutoAssigner
	err = d.db.SelectContext(ctx, &list, query, args...)
	return list, errors.Wrap(err, "list autoassigners")
}

type cursorNotFoundErr struct{}

func (e cursorNotFoundErr) Error() string {
	return "cursor not found"
}

func (e cursorNotFoundErr) NotFound() bool {
	return true
}
