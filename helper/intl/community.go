//go:build !professional
// +build !professional

package intl

import (
	"embed"
)

//go:embed community/*
var UnifiedFS embed.FS
