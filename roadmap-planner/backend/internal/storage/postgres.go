/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package storage

import (
	"database/sql"
	"fmt"

	// jackc/pgx/v5 in stdlib mode registers as the "pgx" driver. We use
	// it through database/sql so the rest of the codebase doesn't need to
	// know it's Postgres.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// OpenPostgres returns a Store backed by a Postgres database.
//
// dsn accepts both URL-style ("postgres://user:pass@host/db?sslmode=…")
// and key-value style ("host=… user=… password=… dbname=… sslmode=…");
// pgx normalises both.
//
// Recommended pool sizing for our workload (one writer goroutine, ~10
// concurrent HTTP readers):
//   - MaxOpenConns: 10
//   - MaxIdleConns: 4
//
// We do not call any extension-creating SQL — the schema is written to
// match what SQLite emits, so a stock Postgres is sufficient. If we ever
// adopt JSONB indexes on `components` / `versions`, that goes in a new
// migration that runs only on the postgres dialect.
func OpenPostgres(dsn string) (Store, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres dsn is empty")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(4)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &genericStore{db: db, d: postgresDialect{}}, nil
}
