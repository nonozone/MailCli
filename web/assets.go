package webassets

import "embed"

// Files contains the static browser shell served by `mailcli web`.
//
//go:embed index.html styles.css app.js
var Files embed.FS
