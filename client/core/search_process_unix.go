//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package core

import (
	"errors"

	"golang.org/x/sys/unix"
)

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := unix.Kill(pid, 0)
	return err == nil || errors.Is(err, unix.EPERM)
}
