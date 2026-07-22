//go:build windows

package rpc

import "golang.org/x/sys/windows"

func availableDiskBytes(path string) (uint64, error) {
	directoryName, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var available uint64
	if err := windows.GetDiskFreeSpaceEx(directoryName, &available, nil, nil); err != nil {
		return 0, err
	}
	return available, nil
}
