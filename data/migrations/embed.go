// Package migrations embeds schema needed by database maintenance tools.
package migrations

import "embed"

//go:embed *.sql
var Files embed.FS
