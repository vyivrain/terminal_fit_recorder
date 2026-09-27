package migrations

import "embed"

// Files contains every database migration so installed single-file binaries
// can initialize and upgrade their database from any working directory.
//
//go:embed *.sql
var Files embed.FS
