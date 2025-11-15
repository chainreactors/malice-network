//go:build professional
// +build professional

package intl

import (
	"embed"
)

//go:embed community/main.lua
//go:embed community/mal.yaml
//go:embed community/modules
//go:embed community/resources
//go:embed professional/main.lua
//go:embed professional/mal.yaml
//go:embed professional/modules
//go:embed professional/resources
var UnifiedFS embed.FS
