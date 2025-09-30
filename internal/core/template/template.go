package template

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Engine struct {
	values map[string]string
}

func NewEngine(values map[string]string) *Engine {
	return &Engine{
		values: values,
	}
}

func (e *Engine) WriteEnvFile(packageDir string) error {
	envPath := filepath.Join(packageDir, ".env")

	lines := make([]string, 0, len(e.values))
	keys := make([]string, 0, len(e.values))
	for k := range e.values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := e.values[k]
		if strings.Contains(v, "\n") || strings.Contains(v, "\"") {
			v = fmt.Sprintf("%q", v)
		}
		lines = append(lines, fmt.Sprintf("%s=%s", k, v))
	}

	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}

	return os.WriteFile(envPath, []byte(content), 0o600)
}

func (e *Engine) SetEnvironment() func() {
	originalEnv := make(map[string]string)

	for key, value := range e.values {
		if original, exists := os.LookupEnv(key); exists {
			originalEnv[key] = original
		}
		if err := os.Setenv(key, value); err != nil {
			continue
		}
	}

	return func() {
		for key := range e.values {
			if original, exists := originalEnv[key]; exists {
				if err := os.Setenv(key, original); err != nil {
					continue
				}
			} else {
				if err := os.Unsetenv(key); err != nil {
					continue
				}
			}
		}
	}
}
