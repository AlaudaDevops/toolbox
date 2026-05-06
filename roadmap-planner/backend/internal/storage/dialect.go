/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package storage

import (
	"fmt"
	"strings"
)

// Dialect captures the differences between SQLite and Postgres at the
// SQL-string level.
//
// We deliberately keep this small. The schema (DDL) and the queries we
// hand-write are 95% portable between the two; the gaps are:
//
//   1. placeholder syntax — SQLite is `?`, Postgres is `$1, $2, …`.
//   2. "round timestamp down to its Monday-week-start DATE" — SQLite
//      uses `DATE(col, 'weekday 1', '-7 days')`, Postgres uses
//      `date_trunc('week', col)::date`.
//
// The remaining differences (JSON column types, foreign-key enforcement,
// pragmas vs. settings) are absorbed at Open* time, not in queries.
type Dialect interface {
	// Name is "sqlite" | "postgres".
	Name() string

	// DriverName matches the database/sql driver registered name. For
	// SQLite that's "sqlite" (modernc.org); for Postgres "pgx" (jackc).
	DriverName() string

	// Placeholder returns the n-th positional placeholder, 1-indexed.
	Placeholder(n int) string

	// WeekStart returns a SQL expression that converts a TIMESTAMP
	// expression to the Monday-00:00 DATE of its week. The expression
	// substitutes col verbatim; pass a fully qualified column name or
	// inline expression.
	WeekStart(col string) string
}

// rebind rewrites a SQL string written with `?` placeholders into the
// dialect's actual placeholder syntax. Walking the string is fine — none
// of our queries are large enough for this to matter.
//
// Naive: it does not handle `?` inside string literals. None of our
// queries embed literal `?` characters; if that ever changes, consider
// using sqlx's parser.
func rebind(d Dialect, q string) string {
	if d.Placeholder(1) == "?" {
		return q
	}
	var b strings.Builder
	b.Grow(len(q) + 16)
	n := 1
	for _, ch := range q {
		if ch == '?' {
			b.WriteString(d.Placeholder(n))
			n++
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

// ---------- SQLite ----------

type sqliteDialect struct{}

func (sqliteDialect) Name() string         { return "sqlite" }
func (sqliteDialect) DriverName() string   { return "sqlite" }
func (sqliteDialect) Placeholder(int) string {
	return "?"
}
func (sqliteDialect) WeekStart(col string) string {
	// SQLite has no native week-start, but `weekday 1` advances to next
	// Monday and `-7 days` shifts back, yielding the *previous* Monday.
	return fmt.Sprintf(`DATE(%s, 'weekday 1', '-7 days')`, col)
}

// ---------- Postgres ----------

type postgresDialect struct{}

func (postgresDialect) Name() string       { return "postgres" }
func (postgresDialect) DriverName() string { return "pgx" }
func (postgresDialect) Placeholder(n int) string {
	return fmt.Sprintf("$%d", n)
}
func (postgresDialect) WeekStart(col string) string {
	// `date_trunc('week', …)` returns the Monday-00:00 timestamp; cast
	// to date to match SQLite's return type.
	return fmt.Sprintf(`date_trunc('week', %s)::date`, col)
}
