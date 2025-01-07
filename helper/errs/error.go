package errs

import (
	"fmt"
)

func Newf(err error, format string, s ...interface{}) error {
	return fmt.Errorf("%w, "+format, s...)
}
