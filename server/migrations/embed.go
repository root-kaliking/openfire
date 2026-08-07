// Package migrations embeds the SQL migration files so they can be executed at
// startup by the store package.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
