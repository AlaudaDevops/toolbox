/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	out := make([]migration, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		// expected: NNNN_name.sql
		parts := strings.SplitN(name, "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("malformed migration filename %q", name)
		}
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("malformed migration version in %q: %w", name, err)
		}
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return nil, err
		}
		out = append(out, migration{version: v, name: name, sql: string(body)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// runMigrations applies all pending migrations on db. Idempotent.
//
// We split each file on `;` followed by EOL because database/sql drivers
// vary in how they handle multi-statement Exec. The DDL itself is
// portable between SQLite and Postgres (see migrations/0001_init.sql);
// only the bookkeeping insert into schema_migrations needs dialect-aware
// placeholder rewriting.
func runMigrations(ctx context.Context, db *sql.DB, d Dialect) error {
	migs, err := loadMigrations()
	if err != nil {
		return err
	}

	// Bootstrap schema_migrations table independently — it's created by the
	// first migration, but we need it before applying it. Idempotent CREATE.
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL)`); err != nil {
		return fmt.Errorf("bootstrap schema_migrations: %w", err)
	}

	applied := map[int]bool{}
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("select applied: %w", err)
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	rows.Close()

	for _, m := range migs {
		if applied[m.version] {
			continue
		}
		if err := applyMigration(ctx, db, d, m); err != nil {
			return fmt.Errorf("apply %s: %w", m.name, err)
		}
	}
	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, d Dialect, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range splitStatements(m.sql) {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("statement %q: %w", firstLine(stmt), err)
		}
	}
	insertSQL := rebind(d, `INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)`)
	if _, err := tx.ExecContext(ctx, insertSQL,
		m.version, time.Now().UTC()); err != nil {
		return err
	}
	return tx.Commit()
}

// splitStatements is a naive splitter for our migration files. It drops
// comments and splits on `;` at end of line. Good enough for the
// hand-written DDL we ship; we'll switch to a real parser if migrations
// grow to need triggers / DO blocks.
func splitStatements(sqlText string) []string {
	lines := strings.Split(sqlText, "\n")
	var buf strings.Builder
	out := []string{}
	for _, l := range lines {
		trim := strings.TrimSpace(l)
		if strings.HasPrefix(trim, "--") || trim == "" {
			continue
		}
		buf.WriteString(l)
		buf.WriteString("\n")
		if strings.HasSuffix(trim, ";") {
			out = append(out, buf.String())
			buf.Reset()
		}
	}
	if strings.TrimSpace(buf.String()) != "" {
		out = append(out, buf.String())
	}
	return out
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
