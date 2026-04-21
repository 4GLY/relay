package migrations

import "embed"

// Files contains the ordered SQL migrations shipped with Relay.
//
//go:embed *.sql
var Files embed.FS
