/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package storage

import (
	"strings"
	"testing"
)

// TestRebindSQLite is the no-op case — `?` placeholders pass through.
func TestRebindSQLite(t *testing.T) {
	in := "INSERT INTO t (a, b, c) VALUES (?, ?, ?)"
	got := rebind(sqliteDialect{}, in)
	if got != in {
		t.Fatalf("sqlite rebind altered input: %q", got)
	}
}

// TestRebindPostgres converts `?` → `$N`, preserving non-placeholder
// punctuation and respecting positional ordering.
func TestRebindPostgres(t *testing.T) {
	cases := []struct{ in, want string }{
		{"SELECT 1", "SELECT 1"},
		{"INSERT INTO t (a, b) VALUES (?, ?)", "INSERT INTO t (a, b) VALUES ($1, $2)"},
		{"WHERE x = ? AND y IN (?, ?, ?)", "WHERE x = $1 AND y IN ($2, $3, $4)"},
	}
	for _, tc := range cases {
		got := rebind(postgresDialect{}, tc.in)
		if got != tc.want {
			t.Fatalf("rebind(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestWeekStartExpressions verify the dialect-specific SQL fragments
// expose the column name they were given (callers compose them into
// larger queries; surface area is the substring check below).
func TestWeekStartExpressions(t *testing.T) {
	if got := (sqliteDialect{}).WeekStart("merged_at"); !strings.Contains(got, "merged_at") || !strings.Contains(got, "weekday 1") {
		t.Fatalf("sqlite WeekStart unexpected: %q", got)
	}
	if got := (postgresDialect{}).WeekStart("merged_at"); !strings.Contains(got, "merged_at") || !strings.Contains(got, "date_trunc") {
		t.Fatalf("postgres WeekStart unexpected: %q", got)
	}
}
