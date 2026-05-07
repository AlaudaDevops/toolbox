/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package storage

import (
	"database/sql"
	"fmt"

	// modernc.org/sqlite is a pure-Go SQLite driver — no CGO. Important so
	// the existing multi-stage Alpine Docker build stays slim and the
	// binary stays statically linked.
	_ "modernc.org/sqlite"
)

// OpenSQLite returns a Store backed by a SQLite database at path.
//
// Pragmas, set via DSN so they apply to every connection (the connection
// pool is capped to one writer below; pragmas would otherwise be
// per-connection):
//   - journal_mode=WAL — single-writer-many-readers; the collector goroutine
//     writes while HTTP handlers read.
//   - synchronous=NORMAL — durability is not as important to us as
//     throughput; any single lost cycle is reconstructable from Jira.
//   - busy_timeout=5000 — back off short lock collisions instead of failing.
//   - foreign_keys=ON — we declare FKs in the schema; let the DB enforce.
//
// The `_time_format=sqlite` driver flag is critical and easy to miss:
// modernc.org/sqlite's default stores time.Time using Go's String()
// format ("2026-05-04 12:00:00 +0000 UTC"), which SQLite's date/time
// functions cannot parse — DATE(col, 'weekday 1', '-7 days') silently
// returns NULL. With `_time_format=sqlite` the value is stored as
// "2026-05-04 12:00:00+00:00", which all date functions accept and
// time.Time still scans back from cleanly.
func OpenSQLite(path string) (Store, error) {
	if path == "" {
		return nil, fmt.Errorf("storage path is empty")
	}
	dsn := fmt.Sprintf(
		"file:%s?_time_format=sqlite&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite at %s: %w", path, err)
	}
	// Even with WAL, having more than one writable conn breaks SQLite's
	// single-writer model under contention. Cap to one writer; the pool
	// auto-grows for read-only ops.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return &genericStore{db: db, d: sqliteDialect{}}, nil
}
