package core

import "errors"

var (
	ErrNotFoundTask  = errors.New("task not found")
	ErrDisableOutput = errors.New("output disabled")
)
