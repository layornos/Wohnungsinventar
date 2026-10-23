// Package store kapselt den Datenzugriff auf SQLite (ADR-0002).
// Abfragen bleiben bei Standard-SQL, damit ein Wechsel zu PostgreSQL möglich bleibt.
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

// FotoVerzeichnis ist der Ordner im Datenverzeichnis, in dem Fotos als Dateien liegen (ADR-0002).
const FotoVerzeichnis = "fotos"

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db *sql.DB
}

// Open bereitet das Datenverzeichnis vor, öffnet die Datenbank und führt ausstehende Migrationen aus.
func Open(ctx context.Context, datenverzeichnis string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(datenverzeichnis, FotoVerzeichnis), 0o750); err != nil {
		return nil, fmt.Errorf("Datenverzeichnis anlegen: %w", err)
	}
	dsn := "file:" + url.PathEscape(filepath.Join(datenverzeichnis, "inventar.db")) +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrieren(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrieren(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migration (version TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	namen, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(namen)
	for _, name := range namen {
		version := strings.TrimSuffix(filepath.Base(name), ".sql")
		var n int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migration WHERE version = ?`, version).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		inhalt, err := migrations.ReadFile(name)
		if err != nil {
			return err
		}
		err = s.inTransaktion(ctx, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, string(inhalt)); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `INSERT INTO schema_migration (version) VALUES (?)`, version)
			return err
		})
		if err != nil {
			return fmt.Errorf("Migration %s: %w", version, err)
		}
	}
	return nil
}

func (s *Store) inTransaktion(ctx context.Context, fn func(*sql.Tx) error) error {
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

// zeitFormat ist RFC 3339 in UTC mit fester Länge. RFC3339Nano ließe Nullen am Ende weg,
// dann sortierte „…:05Z“ als Text hinter „…:05.5Z“.
const zeitFormat = "2006-01-02T15:04:05.000000000Z07:00"

// Zeitpunkte werden als Text gespeichert, damit sie sich textuell vergleichen und sortieren lassen.
func zeitText(t time.Time) string { return t.UTC().Format(zeitFormat) }

func zeitLesen(s string) (time.Time, error) { return time.Parse(time.RFC3339Nano, s) }
