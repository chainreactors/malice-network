//go:build professional
// +build professional

package consts

import "embed"

//go:embed professional/* custom/* community/*
var UnifiedFS embed.FS

//go:embed professional.yaml
var DefaultProfile []byte

var DefaultRDI = RDIMutant
