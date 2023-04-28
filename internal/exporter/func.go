package exporter

import (
	"fmt"
	"os"
)

// mkdir checks if the provided path exists and creates it if it does not.
func mkdir(pth string) error {
	// Check if the directory already exists.
	if _, err := os.Stat(pth); os.IsNotExist(err) {
		// Create the directory if it does not exist.
		if err = os.MkdirAll(pth, os.ModePerm); err != nil {
			return fmt.Errorf("os.MkdirAll: %w", err)
		}
	}

	return nil
}
