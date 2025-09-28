package config

import (
	"os"
	"path/filepath"
)

func GetStateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".compak"), nil
}
