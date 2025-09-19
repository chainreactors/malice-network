//go:build professional
// +build professional

package intl

import (
	"embed"
)

//go:embed professional/* custom/* community/*
var UnifiedFS embed.FS
