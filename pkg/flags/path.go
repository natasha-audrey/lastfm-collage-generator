package flags

import (
	"fmt"
	"os"
)

func parsePath(path string) (string, error) {
	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	// Attempt to create it
	var d []byte
	if err := os.WriteFile(path, d, 0644); err == nil {
		os.Remove(path) // And delete it
		return path, nil
	}

	return "", fmt.Errorf("path %s can't be created", path)
}
