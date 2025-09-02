//go:build !professional
// +build !professional

package consts

import _ "embed"

//go:embed community.yaml
var DefaultProfile []byte

var DefaultRDI = RDIDonut
