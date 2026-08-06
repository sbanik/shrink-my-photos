//go:build darwin || linux

package storage

import "syscall"

// AvailableSpace returns bytes available to the current user on the filesystem.
func AvailableSpace(path string) (int64, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return 0, err
	}
	return int64(stats.Bavail) * int64(stats.Bsize), nil
}
