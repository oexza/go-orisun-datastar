package appdb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	toolbeltdb "github.com/delaneyj/toolbelt/db"
	appmigrations "github.com/example/hono-event-starter-go/migrations"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

var (
	ErrNoRows   = errors.New("no rows in result set")
	placeholder = regexp.MustCompile(`\$(\d+)`)
)

type DB struct {
	inner *toolbeltdb.Database
}

type Result struct{}

func Open(ctx context.Context, filename string) (*DB, error) {
	migrations, err := appmigrations.SQL()
	if err != nil {
		return nil, err
	}
	inner, err := toolbeltdb.NewDatabase(ctx,
		toolbeltdb.DatabaseWithFilename(filename),
		toolbeltdb.DatabaseWithMigrations(migrations),
		toolbeltdb.DatabaseWithPragmas("foreign_keys = ON", "busy_timeout = 5000"),
	)
	if err != nil {
		return nil, err
	}
	return &DB{inner: inner}, nil
}

func (db *DB) Close() error {
	if db == nil || db.inner == nil {
		return nil
	}
	return db.inner.Close()
}

func (db *DB) Exec(ctx context.Context, query string, args ...any) (Result, error) {
	err := db.inner.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return sqlitex.Execute(conn, normalizeSQL(query), &sqlitex.ExecOptions{Args: normalizeArgs(args)})
	})
	return Result{}, err
}

func (db *DB) Query(ctx context.Context, query string, args ...any) (*Rows, error) {
	rows := &Rows{}
	err := db.inner.ReadTX(ctx, func(conn *sqlite.Conn) error {
		return sqlitex.Execute(conn, normalizeSQL(query), &sqlitex.ExecOptions{
			Args: normalizeArgs(args),
			ResultFunc: func(stmt *sqlite.Stmt) error {
				row := make([]string, stmt.ColumnCount())
				for i := range row {
					row[i] = stmt.ColumnText(i)
				}
				rows.rows = append(rows.rows, row)
				return nil
			},
		})
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (db *DB) QueryRow(ctx context.Context, query string, args ...any) Row {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return Row{err: err}
	}
	if len(rows.rows) == 0 {
		return Row{err: ErrNoRows}
	}
	return Row{values: rows.rows[0]}
}

type Rows struct {
	rows   [][]string
	index  int
	closed bool
}

func (r *Rows) Next() bool {
	if r.closed || r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *Rows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.rows) {
		return ErrNoRows
	}
	return scanValues(r.rows[r.index-1], dest...)
}

func (r *Rows) Close() {
	r.closed = true
}

func (r *Rows) Err() error {
	return nil
}

type Row struct {
	values []string
	err    error
}

func (r Row) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return scanValues(r.values, dest...)
}

func scanValues(values []string, dest ...any) error {
	if len(dest) > len(values) {
		return fmt.Errorf("scan destination count %d exceeds value count %d", len(dest), len(values))
	}
	for i, target := range dest {
		if err := scanValue(values[i], target); err != nil {
			return fmt.Errorf("scan column %d: %w", i, err)
		}
	}
	return nil
}

func scanValue(value string, target any) error {
	switch out := target.(type) {
	case *string:
		*out = value
	case *bool:
		*out = value == "1" || strings.EqualFold(value, "true")
	case *int:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		*out = parsed
	case *int64:
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		*out = parsed
	case *time.Time:
		parsed, err := parseTime(value)
		if err != nil {
			return err
		}
		*out = parsed
	default:
		return fmt.Errorf("unsupported scan target %s", reflect.TypeOf(target))
	}
	return nil
}

func parseTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time %q", value)
}

func normalizeSQL(query string) string {
	query = strings.ReplaceAll(query, "::bigint", "")
	query = strings.ReplaceAll(query, "now()", "CURRENT_TIMESTAMP")
	return placeholder.ReplaceAllString(query, "?$1")
}

func normalizeArgs(args []any) []any {
	out := make([]any, len(args))
	for i, arg := range args {
		switch value := arg.(type) {
		case time.Time:
			out[i] = value.UTC().Format("2006-01-02 15:04:05")
		default:
			out[i] = value
		}
	}
	return out
}
