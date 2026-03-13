package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SecretRef references a secret from a configured source.
// Inspired by openclaw's SecretRefSource pattern.
type SecretRef struct {
	Source  string `yaml:"source"`  // "env", "file", or "exec"
	ID      string `yaml:"id"`      // env var name or file path
	Command string `yaml:"command"` // command to execute (for exec source)
}

// SecretInput can be either a plain string or a SecretRef.
type SecretInput struct {
	Value string
	Ref   *SecretRef
}

func (s *SecretInput) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try plain string first
	var str string
	if err := unmarshal(&str); err == nil {
		s.Value = str
		return nil
	}

	// Try SecretRef
	var ref SecretRef
	if err := unmarshal(&ref); err == nil && ref.Source != "" {
		s.Ref = &ref
		return nil
	}

	return fmt.Errorf("invalid secret input: must be a string or {source, id/command}")
}

// Resolve returns the secret value by reading from the configured source.
func (s *SecretInput) Resolve() (string, error) {
	if s == nil {
		return "", nil
	}

	// Plain string (already resolved, possibly via env expansion)
	if s.Ref == nil {
		return s.Value, nil
	}

	switch s.Ref.Source {
	case "env":
		id := s.Ref.ID
		if id == "" {
			return "", fmt.Errorf("env secret ref requires 'id'")
		}
		val := os.Getenv(id)
		if val == "" {
			return "", fmt.Errorf("environment variable %q is not set", id)
		}
		return val, nil

	case "file":
		id := s.Ref.ID
		if id == "" {
			return "", fmt.Errorf("file secret ref requires 'id' (file path)")
		}
		data, err := os.ReadFile(id)
		if err != nil {
			return "", fmt.Errorf("failed to read secret file %q: %w", id, err)
		}
		return strings.TrimSpace(string(data)), nil

	case "exec":
		command := s.Ref.Command
		if command == "" {
			return "", fmt.Errorf("exec secret ref requires 'command'")
		}
		parts := strings.Fields(command)
		cmd := exec.Command(parts[0], parts[1:]...)
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to execute secret command %q: %w", command, err)
		}
		return strings.TrimSpace(string(out)), nil

	default:
		return "", fmt.Errorf("unknown secret source %q (supported: env, file, exec)", s.Ref.Source)
	}
}
