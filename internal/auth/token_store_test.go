package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTokenStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	store := NewTokenStore(path)

	token := &CachedToken{
		AccessToken: "gho_abc123",
		GitHubUser:  "testuser",
		SavedAt:     time.Now().UTC().Truncate(time.Second),
	}

	if err := store.Save(token); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.AccessToken != token.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, token.AccessToken)
	}
	if loaded.GitHubUser != token.GitHubUser {
		t.Errorf("GitHubUser = %q, want %q", loaded.GitHubUser, token.GitHubUser)
	}
}

func TestTokenStore_LoadMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	store := NewTokenStore(path)

	token, err := store.Load()
	if err != nil {
		t.Fatalf("Load should not error for missing file: %v", err)
	}
	if token != nil {
		t.Error("expected nil token for missing file")
	}
}

func TestTokenStore_Clear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	store := NewTokenStore(path)

	_ = store.Save(&CachedToken{AccessToken: "gho_abc123", GitHubUser: "testuser", SavedAt: time.Now()})

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	token, _ := store.Load()
	if token != nil {
		t.Error("expected nil token after clear")
	}
}

func TestTokenStore_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	store := NewTokenStore(path)

	_ = store.Save(&CachedToken{AccessToken: "gho_abc123", GitHubUser: "testuser", SavedAt: time.Now()})

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permission = %o, want 0600", perm)
	}
}
