//go:build professional
// +build professional

package consts

import _ "embed"

//go:embed professional.yaml
var DefaultProfile []byte

var DefaultRDI = RDIMutant
