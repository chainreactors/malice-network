//go:build !professional
// +build !professional

package intl

import (
	"embed"
)

//go:embed community/main.lua
//go:embed community/mal.yaml
//go:embed community/modules
//go:embed community/resources
var UnifiedFS embed.FS
