package utils

import (
	"os"
)

func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func WriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
