//go:build !professional
// +build !professional

package consts

import (
	"embed"
)

//go:embed community.yaml
var DefaultProfile []byte

//go:embed community/*
var UnifiedFS embed.FS

var DefaultRDI = RDIDonut
