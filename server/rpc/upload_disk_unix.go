//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package rpc

import "golang.org/x/sys/unix"

func availableDiskBytes(path string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}
	blockSize := uint64(stat.Bsize)
	if blockSize == 0 || stat.Bavail > ^uint64(0)/blockSize {
		return ^uint64(0), nil
	}
	return stat.Bavail * blockSize, nil
}
