// Package store encapsulates data access on SQLite (ADR-0002).
// Queries stick to standard SQL so that a switch to PostgreSQL stays possible.
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// PhotoDir is the folder in the data directory where photos are stored as files (ADR-0002).
// The name stays German so existing data directories keep working.
const PhotoDir = "fotos"

// DBFile is the name of the SQLite database in the data directory.
const DBFile = "inventar.db"

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db *sql.DB
}

// Open prepares the data directory, opens the database and runs pending migrations.
func Open(ctx context.Context, dataDir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dataDir, PhotoDir), 0o750); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	dsn := "file:" + url.PathEscape(filepath.Join(dataDir, DBFile)) +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Snapshot writes a consistent copy of the database to path with VACUUM INTO. It is safe while
// the server is running; path must not exist yet.
func (s *Store) Snapshot(ctx context.Context, path string) error {
	_, err := s.db.ExecContext(ctx, "VACUUM INTO ?", path)
	return err
}

// migrate runs pending migrations on one connection with foreign keys switched off, so that a
// migration can rebuild a table others refer to (SQLite's procedure for schema changes).
// Before each commit foreign_key_check must come back empty.
func (s *Store) migrate(ctx context.Context) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migration (version TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	// foreign_keys cannot be changed inside a transaction. With legacy_alter_table off, renaming a
	// table also rewrites the references to it in other tables.
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), `PRAGMA foreign_keys = ON`)
	if _, err := conn.ExecContext(ctx, `PRAGMA legacy_alter_table = OFF`); err != nil {
		return err
	}
	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, name := range names {
		version := strings.TrimSuffix(filepath.Base(name), ".sql")
		var n int
		if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migration WHERE version = ?`, version).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		content, err := migrations.ReadFile(name)
		if err != nil {
			return err
		}
		if err := runMigration(ctx, conn, version, string(content)); err != nil {
			return fmt.Errorf("migration %s: %w", version, err)
		}
	}
	return nil
}

func runMigration(ctx context.Context, conn *sql.Conn, version, content string) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, content); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return err
	}
	violated := rows.Next()
	rows.Close()
	if violated {
		return fmt.Errorf("foreign key check failed")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migration (version) VALUES (?)`, version); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) inTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// timeFormat is RFC 3339 in UTC with a fixed length. RFC3339Nano would drop trailing zeros,
// and then "…:05Z" would sort as text after "…:05.5Z".
const timeFormat = "2006-01-02T15:04:05.000000000Z07:00"

// Timestamps are stored as text so they compare and sort as text.
func timeText(t time.Time) string { return t.UTC().Format(timeFormat) }

func parseTime(s string) (time.Time, error) { return time.Parse(time.RFC3339Nano, s) }
