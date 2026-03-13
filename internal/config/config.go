package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	Server         ServerConfig `yaml:"server"`
	GitHub         GitHubConfig `yaml:"github"`
	AllowedActions []string     `yaml:"allowed_actions"`
	Audit          AuditConfig  `yaml:"audit"`
}

type ServerConfig struct {
	Port    int    `yaml:"port"`
	BaseURL string `yaml:"base_url"`
}

type GitHubConfig struct {
	ClientID string `yaml:"client_id"`
}

type AuditConfig struct {
	DBPath string `yaml:"db_path"`
}

// Load reads and parses the config file, expanding environment variables.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, err
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Audit.DBPath == "" {
		cfg.Audit.DBPath = "./audit.db"
	}
	return &cfg, nil
}

// IsActionAllowed checks if the given action is in the allowlist.
func (c *Config) IsActionAllowed(action string) bool {
	for _, a := range c.AllowedActions {
		if strings.EqualFold(a, action) {
			return true
		}
	}
	return false
}
