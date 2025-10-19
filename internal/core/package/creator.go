package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Creator struct{}

func NewCreator() (*Creator, error) {
	return &Creator{}, nil
}

func New() (*Creator, error) {
	return &Creator{}, nil
}

type PackageTemplate struct {
	Name        string
	Version     string
	Description string
	Author      string
	Homepage    string
	Repository  string
	Source      string
	Parameters  map[string]ParameterTemplate
}

type ParameterTemplate struct {
	Type        string
	Default     string
	Required    bool
	Description string
}

func (c *Creator) AnalyzeComposeFile(composePath string) (template *PackageTemplate, err error) {
	projectDir := filepath.Dir(composePath)
	composeFilename := filepath.Base(composePath)

	root, err := os.OpenRoot(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open project directory: %w", err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	file, err := root.Open(composeFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to open compose file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	rawContent, err := os.ReadFile(file.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	variables := c.extractVariablesFromRawYAML(string(rawContent))

	parameters := make(map[string]ParameterTemplate)
	for varName := range variables {
		parameters[varName] = ParameterTemplate{
			Type:        c.inferParameterType(varName),
			Default:     "",
			Required:    false,
			Description: c.generateDescription(varName),
		}
	}

	packageName := filepath.Base(projectDir)
	packageName = strings.ToLower(packageName)

	template = &PackageTemplate{
		Name:        packageName,
		Version:     "1.0.0",
		Description: fmt.Sprintf("%s Docker Compose application", packageName),
		Parameters:  parameters,
	}
	return template, nil
}

func (c *Creator) extractVariablesFromRawYAML(content string) map[string]bool {
	variables := make(map[string]bool)

	varPattern := regexp.MustCompile(`\$\{?([A-Z0-9_]+)(?::-[^}]*)?\}?`)

	matches := varPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if !c.isComposeBuiltinVar(varName) {
				variables[varName] = true
			}
		}
	}

	return variables
}

func (c *Creator) isComposeBuiltinVar(varName string) bool {
	builtins := map[string]bool{
		"COMPOSE_PROJECT_NAME":          true,
		"COMPOSE_FILE":                  true,
		"COMPOSE_PATH_SEPARATOR":        true,
		"COMPOSE_HTTP_TIMEOUT":          true,
		"COMPOSE_TLS_VERSION":           true,
		"COMPOSE_CONVERT_WINDOWS_PATHS": true,
		"DOCKER_CERT_PATH":              true,
		"DOCKER_HOST":                   true,
		"DOCKER_TLS_VERIFY":             true,
	}
	return builtins[varName]
}

func (c *Creator) inferParameterType(varName string) string {
	lower := strings.ToLower(varName)

	typeRules := []struct {
		keywords  []string
		paramType string
	}{
		{[]string{"enable", "disable", "debug"}, "boolean"},
		{[]string{"port", "count", "size", "limit", "timeout", "replicas", "workers"}, "number"},
	}

	for _, rule := range typeRules {
		for _, keyword := range rule.keywords {
			if strings.Contains(lower, keyword) {
				return rule.paramType
			}
		}
	}

	return "string"
}

func (c *Creator) generateDescription(varName string) string {
	words := strings.Split(strings.ToLower(varName), "_")
	for i, word := range words {
		if word != "" {
			words[i] = strings.ToUpper(string(word[0])) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func (c *Creator) GeneratePackageYAML(template *PackageTemplate) (string, error) {
	type orderedParam struct {
		Type        string `yaml:"type"`
		Description string `yaml:"description"`
		Default     string `yaml:"default"`
	}

	type orderedPackage struct {
		Name        string                  `yaml:"name"`
		Version     string                  `yaml:"version"`
		Description string                  `yaml:"description"`
		Author      string                  `yaml:"author"`
		Homepage    string                  `yaml:"homepage"`
		Repository  string                  `yaml:"repository"`
		Source      string                  `yaml:"source"`
		Parameters  map[string]orderedParam `yaml:"parameters,omitempty"`
	}

	params := make(map[string]orderedParam)
	for name, param := range template.Parameters {
		params[name] = orderedParam{
			Type:        param.Type,
			Description: "",
			Default:     "",
		}
	}

	pkg := orderedPackage{
		Name:        template.Name,
		Version:     template.Version,
		Description: "",
		Author:      "",
		Homepage:    "",
		Repository:  "",
		Source:      "",
		Parameters:  params,
	}

	data, err := yaml.Marshal(pkg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal package.yaml: %w", err)
	}

	return string(data), nil
}

func (c *Creator) ReadExistingEnvFile(envPath string) (envVars map[string]string, err error) {
	envDir := filepath.Dir(envPath)
	envFilename := filepath.Base(envPath)

	root, err := os.OpenRoot(envDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open directory: %w", err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	file, err := root.Open(envFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	envVars, err = godotenv.Read(file.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to parse env file: %w", err)
	}

	return envVars, nil
}
