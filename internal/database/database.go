package database

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		var applied bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, entry.Name()).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		for _, statement := range splitStatements(string(body)) {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err = tx.Exec(ctx, statement); err != nil {
				_ = tx.Rollback(ctx)
				return fmt.Errorf("apply %s: %w", entry.Name(), err)
			}
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, entry.Name()); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record %s: %w", entry.Name(), err)
		}
		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// splitStatements separates SQL commands without splitting semicolons inside
// quoted strings, comments, or PostgreSQL dollar-quoted blocks.
func splitStatements(sql string) []string {
	var statements []string
	start := 0
	var dollarTag string
	var quote byte
	lineComment := false
	blockComment := false

	for i := 0; i < len(sql); i++ {
		if lineComment {
			if sql[i] == '\n' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if i+1 < len(sql) && sql[i] == '*' && sql[i+1] == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if dollarTag != "" {
			if strings.HasPrefix(sql[i:], dollarTag) {
				i += len(dollarTag) - 1
				dollarTag = ""
			}
			continue
		}
		if quote != 0 {
			if sql[i] == quote {
				if i+1 < len(sql) && sql[i+1] == quote {
					i++
				} else {
					quote = 0
				}
			} else if sql[i] == '\\' && quote == '\'' && i+1 < len(sql) {
				i++
			}
			continue
		}

		switch {
		case i+1 < len(sql) && sql[i:i+2] == "--":
			lineComment = true
			i++
		case i+1 < len(sql) && sql[i:i+2] == "/*":
			blockComment = true
			i++
		case sql[i] == '\'' || sql[i] == '"':
			quote = sql[i]
		case sql[i] == '$':
			for end := i + 1; end < len(sql); end++ {
				if sql[end] == '$' {
					dollarTag = sql[i : end+1]
					i = end
					break
				}
				if !(sql[end] == '_' || sql[end] >= 'a' && sql[end] <= 'z' || sql[end] >= 'A' && sql[end] <= 'Z' || end > i+1 && sql[end] >= '0' && sql[end] <= '9') {
					break
				}
			}
		case sql[i] == ';':
			statements = append(statements, sql[start:i])
			start = i + 1
		}
	}
	if tail := strings.TrimSpace(sql[start:]); tail != "" {
		statements = append(statements, sql[start:])
	}
	return statements
}
