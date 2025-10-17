package cli

import (
	"fmt"
	"strings"
)

func validatePackageName(packageName string) error {
	if packageName == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	if strings.ContainsAny(packageName, "/\\") {
		return fmt.Errorf("invalid package name: %q", packageName)
	}
	return nil
}
