package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"

	"goscrapy/internal/logger"
	"goscrapy/internal/timeutil"
)

func Migrate(db *sqlx.DB, dir string) error {
	if dir == "" {
		dir = "migrations"
	}
	if _, err := os.Stat(dir); err != nil {
		// Docker image copies migrations to /app/migrations.
		if alt := "/app/migrations"; alt != dir {
			if _, err2 := os.Stat(alt); err2 == nil {
				dir = alt
			} else {
				return fmt.Errorf("migrations dir %s: %w", dir, err)
			}
		} else {
			return fmt.Errorf("migrations dir %s: %w", dir, err)
		}
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL
	)`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	applied := map[string]struct{}{}
	rows, err := db.Queryx(`SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("list applied migrations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return err
		}
		applied[v] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	lg := logger.Named("migrate")
	for _, name := range names {
		if _, ok := applied[name]; ok {
			continue
		}
		path := filepath.Join(dir, name)
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		tx, err := db.Beginx()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(version, applied_at) VALUES($1,$2)`, name, timeutil.NowNaive()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		lg.Info("applied migration")
		_ = name
	}
	return nil
}
