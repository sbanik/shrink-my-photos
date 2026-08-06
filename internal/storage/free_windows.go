//go:build windows

package storage

import "golang.org/x/sys/windows"

// AvailableSpace returns bytes available to the current user on the filesystem.
func AvailableSpace(path string) (int64, error) {
	var available, total, free uint64
	err := windows.GetDiskFreeSpaceEx(windows.StringToUTF16Ptr(path), &available, &total, &free)
	return int64(available), err
}
