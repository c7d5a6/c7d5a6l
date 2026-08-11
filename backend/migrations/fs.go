package migrations

import "embed"

// FS holds numbered *.sql migration files (e.g. 001_init.sql).
//
//go:embed *.sql
var FS embed.FS
