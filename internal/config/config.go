package config

import (
	"fmt"
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
	ClientID     SecretInput `yaml:"client_id"`
	ClientSecret SecretInput `yaml:"client_secret"`
}

type AuditConfig struct {
	DBPath string `yaml:"db_path"`
}

// ResolvedConfig holds config with all secrets resolved to plain strings.
type ResolvedConfig struct {
	Server         ServerConfig
	GitHub         ResolvedGitHubConfig
	AllowedActions []string
	Audit          AuditConfig
}

type ResolvedGitHubConfig struct {
	ClientID     string
	ClientSecret string
}

// Load reads and parses the config file, expanding environment variables.
// If the config file does not exist, it falls back to a default config
// populated from environment variables.
func Load(path string) (*ResolvedConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultResolvedConfig()
		}
		return nil, err
	}

	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, err
	}

	return cfg.Resolve()
}

func defaultResolvedConfig() (*ResolvedConfig, error) {
	return &ResolvedConfig{
		Server: ServerConfig{
			Port:    9091,
			BaseURL: "http://127.0.0.1:9091",
		},
		GitHub: ResolvedGitHubConfig{
			ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
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
	}, nil
}

// Resolve resolves all secrets and returns a ResolvedConfig.
func (c *Config) Resolve() (*ResolvedConfig, error) {
	clientID, err := c.GitHub.ClientID.Resolve()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve github.client_id: %w", err)
	}

	clientSecret, err := c.GitHub.ClientSecret.Resolve()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve github.client_secret: %w", err)
	}

	rc := &ResolvedConfig{
		Server:         c.Server,
		GitHub:         ResolvedGitHubConfig{ClientID: clientID, ClientSecret: clientSecret},
		AllowedActions: c.AllowedActions,
		Audit:          c.Audit,
	}

	applyDefaults(rc)
	return rc, nil
}

func applyDefaults(cfg *ResolvedConfig) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 9091
	}
	if cfg.Server.BaseURL == "" {
		cfg.Server.BaseURL = "http://127.0.0.1:9091"
	}
	if cfg.GitHub.ClientID == "" {
		cfg.GitHub.ClientID = os.Getenv("GITHUB_CLIENT_ID")
	}
	if cfg.GitHub.ClientSecret == "" {
		cfg.GitHub.ClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	}
	if cfg.Audit.DBPath == "" {
		cfg.Audit.DBPath = "./audit.db"
	}
}

// IsActionAllowed checks if the given action is in the allowlist.
func (c *ResolvedConfig) IsActionAllowed(action string) bool {
	for _, a := range c.AllowedActions {
		if strings.EqualFold(a, action) {
			return true
		}
	}
	return false
}
