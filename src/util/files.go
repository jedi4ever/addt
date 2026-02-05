package util

import (
	"io/fs"
	"os"
	"path/filepath"
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

// SafeCopyDir copies a directory recursively if it exists
func SafeCopyDir(src, dst string) {
	info, err := os.Stat(src)
	if err != nil || !info.IsDir() {
		return
	}

	if err := os.MkdirAll(dst, 0700); err != nil {
		return
	}

	filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return nil
		}

		dstPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			os.MkdirAll(dstPath, 0700)
		} else {
			SafeCopyFile(path, dstPath)
		}

		return nil
	})
}
