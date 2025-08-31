package errs

import (
	"errors"
)

var (
	ErrNullDomain = errors.New("auto cert requires a domain")
)
