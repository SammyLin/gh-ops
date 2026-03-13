package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_FullConfig(t *testing.T) {
	yaml := `
server:
  port: 9090
  base_url: "https://example.com"
github:
  client_id: "id123"
  client_secret: "secret456"
allowed_actions:
  - "merge"
  - "approve"
audit:
  db_path: "/tmp/audit.db"
`
	cfg, err := Load(writeTempConfig(t, yaml))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Server.BaseURL != "https://example.com" {
		t.Errorf("BaseURL = %q, want https://example.com", cfg.Server.BaseURL)
	}
	if cfg.GitHub.ClientID != "id123" {
		t.Errorf("ClientID = %q, want id123", cfg.GitHub.ClientID)
	}
	if cfg.GitHub.ClientSecret != "secret456" {
		t.Errorf("ClientSecret = %q, want secret456", cfg.GitHub.ClientSecret)
	}
	if cfg.Audit.DBPath != "/tmp/audit.db" {
		t.Errorf("DBPath = %q, want /tmp/audit.db", cfg.Audit.DBPath)
	}
	if len(cfg.AllowedActions) != 2 {
		t.Errorf("AllowedActions len = %d, want 2", len(cfg.AllowedActions))
	}
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load(writeTempConfig(t, "{}"))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server.Port != 9091 {
		t.Errorf("default Port = %d, want 9091", cfg.Server.Port)
	}
	if cfg.Audit.DBPath != "./audit.db" {
		t.Errorf("default DBPath = %q, want ./audit.db", cfg.Audit.DBPath)
	}
}

func TestLoad_EnvExpansion(t *testing.T) {
	t.Setenv("TEST_GH_CLIENT_ID", "env-id")
	yaml := `
github:
  client_id: "$TEST_GH_CLIENT_ID"
`
	cfg, err := Load(writeTempConfig(t, yaml))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.GitHub.ClientID != "env-id" {
		t.Errorf("ClientID = %q, want env-id", cfg.GitHub.ClientID)
	}
}

func TestLoad_FileNotFound_FallsBackToDefaults(t *testing.T) {
	t.Setenv("GITHUB_CLIENT_ID", "env-fallback-id")
	cfg, err := Load("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("Load() should not error for missing file, got: %v", err)
	}
	if cfg.Server.Port != 9091 {
		t.Errorf("default Port = %d, want 9091", cfg.Server.Port)
	}
	if cfg.GitHub.ClientID != "env-fallback-id" {
		t.Errorf("ClientID = %q, want env-fallback-id", cfg.GitHub.ClientID)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	_, err := Load(writeTempConfig(t, ":::invalid"))
	if err == nil {
		t.Fatal("Load() should return error for invalid YAML")
	}
}

func TestIsActionAllowed(t *testing.T) {
	cfg := &ResolvedConfig{AllowedActions: []string{"merge", "approve"}}

	if !cfg.IsActionAllowed("merge") {
		t.Error("merge should be allowed")
	}
	if !cfg.IsActionAllowed("MERGE") {
		t.Error("MERGE (case-insensitive) should be allowed")
	}
	if cfg.IsActionAllowed("delete") {
		t.Error("delete should not be allowed")
	}
}

func TestLoad_SecretRefEnv(t *testing.T) {
	t.Setenv("MY_CLIENT_ID", "from-env-ref")
	yaml := `
github:
  client_id:
    source: env
    id: MY_CLIENT_ID
`
	cfg, err := Load(writeTempConfig(t, yaml))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.GitHub.ClientID != "from-env-ref" {
		t.Errorf("ClientID = %q, want from-env-ref", cfg.GitHub.ClientID)
	}
}

func TestLoad_SecretRefExec(t *testing.T) {
	yaml := `
github:
  client_id:
    source: exec
    command: "echo test-exec-id"
`
	cfg, err := Load(writeTempConfig(t, yaml))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.GitHub.ClientID != "test-exec-id" {
		t.Errorf("ClientID = %q, want test-exec-id", cfg.GitHub.ClientID)
	}
}

func TestLoad_SecretRefFile(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("file-secret-value\n"), 0600); err != nil {
		t.Fatal(err)
	}

	yaml := `
github:
  client_id:
    source: file
    id: "` + secretPath + `"
`
	cfg, err := Load(writeTempConfig(t, yaml))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.GitHub.ClientID != "file-secret-value" {
		t.Errorf("ClientID = %q, want file-secret-value", cfg.GitHub.ClientID)
	}
}
