// Package migrations fornece os ficheiros SQL embebidos usados por golang-migrate.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
