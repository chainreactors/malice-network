package errs

import "errors"

var (
	ErrInvalidStart     = errors.New("read invalid start delimiter")
	ErrInvalidEnd       = errors.New("read invalid end delimiter")
	ErrInvalidMagic     = errors.New("read invalid magic")
	ErrInvalidHeader    = errors.New("invalid header")
	ErrNullSpites       = errors.New("parsed 0 spites")
	ErrInvalidId        = errors.New("invalid session id")
	ErrInvalidImplant   = errors.New("invalid implant")
	ErrPacketTooLarge   = errors.New("packet too large")
	ErrInvalidEncType   = errors.New("invalid encryption type")
	ErrNotFoundPipeline = errors.New("not found pipeline")
)
