// Package store owns app.db (plan §6): migrator plus one
// repository per table group. Writes run in transactions;
// list queries are keyset-paginated (ts, id) — no OFFSET.
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Open opens (creating) app.db at path with WAL + foreign keys.
func Open(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL", path)
	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	db := &DB{db: sqlDB, path: path}
	if err := db.migrate(context.Background()); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

// DB wraps the app database handle.
type DB struct {
	db   *sql.DB
	path string
}

// Close flushes and closes the handle (shutdown path, §8).
func (d *DB) Close() error { return d.db.Close() }

// migrate applies numbered migrations, tracking PRAGMA user_version.
// Never edit a shipped migration.
func (d *DB) migrate(ctx context.Context) error {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	var v int
	if err := d.db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	// WL-6.2: back up before applying any upgrade migration.
	if v > 0 && v < len(names) {
		if err := d.BackupTo(fmt.Sprintf("%s.pre-migrate-v%d", d.path, v)); err != nil {
			return err
		}
	}
	for i, name := range names {
		want := i + 1
		if want <= v {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := d.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version=%d`, want)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Version reports PRAGMA user_version (tests + backup stamp).
func (d *DB) Version(ctx context.Context) (int, error) {
	var v int
	err := d.db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&v)
	return v, err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
