package util

import (
	"os"
)

// SafeCopyFile copies a file if it exists
func SafeCopyFile(src, dst string) {
	if _, err := os.Stat(src); err != nil {
		return
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return
	}
	os.WriteFile(dst, data, 0600)
}
