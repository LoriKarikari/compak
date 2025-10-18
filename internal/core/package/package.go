package pkg

import (
	"time"
)

type Package struct {
	Name        string            `yaml:"name" json:"name"`
	Version     string            `yaml:"version" json:"version"`
	Description string            `yaml:"description" json:"description"`
	Author      string            `yaml:"author" json:"author"`
	License     string            `yaml:"license" json:"license"`
	Homepage    string            `yaml:"homepage" json:"homepage"`
	Repository  string            `yaml:"repository" json:"repository"`
	Source      string            `yaml:"source" json:"source"`
	Parameters  map[string]Param  `yaml:"parameters" json:"parameters"`
	Values      map[string]string `yaml:"values" json:"values"`
}

type Param struct {
	Description string `yaml:"description" json:"description"`
	Type        string `yaml:"type" json:"type"`
	Default     string `yaml:"default" json:"default"`
	Required    bool   `yaml:"required" json:"required"`
}

type InstalledPackage struct {
	Package     Package           `json:"package"`
	InstallTime time.Time         `json:"install_time"`
	Values      map[string]string `json:"values"`
	Status      string            `json:"status"`
}

type Client struct {
	stateDir string
}
