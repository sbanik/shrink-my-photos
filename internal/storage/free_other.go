//go:build !darwin && !linux && !windows

package storage

import "fmt"

func AvailableSpace(path string) (int64, error) {
	return 0, fmt.Errorf("available-space checks are unsupported on this platform")
}
