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
// If the config file does not exist, it falls back to a default config
// populated from environment variables.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultConfig(), nil
		}
		return nil, err
	}

	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, err
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

func defaultConfig() *Config {
	cfg := &Config{
		Server: ServerConfig{
			Port:    9091,
			BaseURL: "http://127.0.0.1:9091",
		},
		GitHub: GitHubConfig{
			ClientID: os.Getenv("GITHUB_CLIENT_ID"),
		},
		AllowedActions: []string{
			"create-repo",
			"merge-pr",
			"create-tag",
			"add-collaborator",
		},
		Audit: AuditConfig{
			DBPath: "./audit.db",
		},
	}
	return cfg
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 9091
	}
	if cfg.Server.BaseURL == "" {
		cfg.Server.BaseURL = "http://127.0.0.1:9091"
	}
	if cfg.GitHub.ClientID == "" {
		cfg.GitHub.ClientID = os.Getenv("GITHUB_CLIENT_ID")
	}
	if cfg.Audit.DBPath == "" {
		cfg.Audit.DBPath = "./audit.db"
	}
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
