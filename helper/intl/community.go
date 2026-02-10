//go:build !professional
// +build !professional

package intl

import (
	"embed"
)

//go:embed community
//go:embed all:custom
var UnifiedFS embed.FS
