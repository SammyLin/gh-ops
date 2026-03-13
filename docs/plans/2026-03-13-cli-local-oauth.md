# CLI Local OAuth Flow Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Transform gh-ops from a long-running web server into a CLI tool that starts a temporary local server, opens the browser for OAuth + action confirmation, executes the action, and exits.

**Architecture:** CLI parses action + params → starts ephemeral localhost HTTP server → shows confirm page in browser → user confirms → GitHub OAuth → execute action → print result to CLI → shutdown server. Token cached locally for reuse (`~/.config/gh-ops/token.json`). The web templates are reused for the confirm/result pages shown in the browser.

**Tech Stack:** Go, net/http (stdlib), cobra (CLI framework), go-github, oauth2, embedded HTML templates, SQLite audit log (unchanged). Removes chi, cors, gorilla/sessions dependencies.

---

## Current State

- `main.go` — entry point with `embed.FS`, `--config` flag, calls `cmd.Run()`
- `cmd/server.go` — long-running HTTP server with chi router
- `internal/auth/oauth.go` — GitHub OAuth with cookie sessions (30-day expiry)
- `internal/actions/handler.go` — 4 GitHub actions (create-repo, merge-pr, create-tag, add-collaborator)
- `internal/config/config.go` — YAML config loader
- `internal/middleware/ratelimit.go` — per-IP rate limiting
- `internal/audit/audit.go` — SQLite audit log
- `web/templates/` — base.html, home.html, result.html, error.html

## Target Flow

```
$ gh-ops create-repo --name my-repo --visibility public

Open this link to authorize:
    http://localhost:18923/confirm

Waiting for authorization...

✅ Repository SammyLin/my-repo created successfully.
```

Browser flow:
1. User clicks link → sees confirm page ("You are about to create repo 'my-repo'. Confirm?")
2. User clicks Confirm → redirects to GitHub OAuth (if no cached token)
3. OAuth callback → action executes → browser shows result page
4. CLI prints result → local server shuts down → process exits

---

### Task 1: Add cobra dependency and scaffold CLI commands

**Files:**
- Modify: `go.mod` (add cobra dependency)
- Create: `cmd/root.go`
- Create: `cmd/create_repo.go`
- Create: `cmd/merge_pr.go`
- Create: `cmd/create_tag.go`
- Create: `cmd/add_collaborator.go`
- Modify: `main.go` (switch from flag-based to cobra)

**Step 1: Add cobra dependency**

Run: `cd /Users/sammylin/CascadeProjects/gh-ops && go get github.com/spf13/cobra@latest`

**Step 2: Create cmd/root.go**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configPath string

var rootCmd = &cobra.Command{
	Use:   "gh-ops",
	Short: "One-click GitHub operations via OAuth",
	Long:  "A CLI tool that executes GitHub operations with user OAuth authorization.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "config file path")
}
```

**Step 3: Create cmd/create_repo.go**

```go
package cmd

import (
	"io/fs"

	"github.com/spf13/cobra"
)

var templateFS fs.FS

func SetTemplateFS(f fs.FS) {
	templateFS = f
}

var createRepoCmd = &cobra.Command{
	Use:   "create-repo",
	Short: "Create a new GitHub repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		visibility, _ := cmd.Flags().GetString("visibility")
		description, _ := cmd.Flags().GetString("description")
		autoInit, _ := cmd.Flags().GetBool("auto-init")

		params := map[string]string{
			"name":       name,
			"visibility": visibility,
			"description": description,
		}
		if !autoInit {
			params["auto_init"] = "false"
		}

		return runAction("create-repo", params)
	},
}

func init() {
	createRepoCmd.Flags().String("name", "", "repository name (required)")
	createRepoCmd.Flags().String("visibility", "public", "public or private")
	createRepoCmd.Flags().String("description", "", "repository description")
	createRepoCmd.Flags().Bool("auto-init", true, "initialize with README")
	_ = createRepoCmd.MarkFlagRequired("name")
	rootCmd.AddCommand(createRepoCmd)
}
```

**Step 4: Create cmd/merge_pr.go**

```go
package cmd

import (
	"github.com/spf13/cobra"
)

var mergePRCmd = &cobra.Command{
	Use:   "merge-pr",
	Short: "Merge a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, _ := cmd.Flags().GetString("repo")
		prNumber, _ := cmd.Flags().GetString("pr-number")
		mergeMethod, _ := cmd.Flags().GetString("merge-method")

		params := map[string]string{
			"repo":         repo,
			"pr_number":    prNumber,
			"merge_method": mergeMethod,
		}

		return runAction("merge-pr", params)
	},
}

func init() {
	mergePRCmd.Flags().String("repo", "", "repository in owner/repo format (required)")
	mergePRCmd.Flags().String("pr-number", "", "pull request number (required)")
	mergePRCmd.Flags().String("merge-method", "merge", "merge, squash, or rebase")
	_ = mergePRCmd.MarkFlagRequired("repo")
	_ = mergePRCmd.MarkFlagRequired("pr-number")
	rootCmd.AddCommand(mergePRCmd)
}
```

**Step 5: Create cmd/create_tag.go**

```go
package cmd

import (
	"github.com/spf13/cobra"
)

var createTagCmd = &cobra.Command{
	Use:   "create-tag",
	Short: "Create a git tag",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, _ := cmd.Flags().GetString("repo")
		tag, _ := cmd.Flags().GetString("tag")
		sha, _ := cmd.Flags().GetString("sha")
		message, _ := cmd.Flags().GetString("message")

		params := map[string]string{
			"repo":    repo,
			"tag":     tag,
			"sha":     sha,
			"message": message,
		}

		return runAction("create-tag", params)
	},
}

func init() {
	createTagCmd.Flags().String("repo", "", "repository in owner/repo format (required)")
	createTagCmd.Flags().String("tag", "", "tag name, e.g. v1.0.0 (required)")
	createTagCmd.Flags().String("sha", "", "commit SHA (default: HEAD of default branch)")
	createTagCmd.Flags().String("message", "", "tag message (creates annotated tag)")
	_ = createTagCmd.MarkFlagRequired("repo")
	_ = createTagCmd.MarkFlagRequired("tag")
	rootCmd.AddCommand(createTagCmd)
}
```

**Step 6: Create cmd/add_collaborator.go**

```go
package cmd

import (
	"github.com/spf13/cobra"
)

var addCollaboratorCmd = &cobra.Command{
	Use:   "add-collaborator",
	Short: "Add a collaborator to a repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, _ := cmd.Flags().GetString("repo")
		user, _ := cmd.Flags().GetString("user")
		permission, _ := cmd.Flags().GetString("permission")

		params := map[string]string{
			"repo":       repo,
			"user":       user,
			"permission": permission,
		}

		return runAction("add-collaborator", params)
	},
}

func init() {
	addCollaboratorCmd.Flags().String("repo", "", "repository in owner/repo format (required)")
	addCollaboratorCmd.Flags().String("user", "", "GitHub username (required)")
	addCollaboratorCmd.Flags().String("permission", "push", "pull, push, or admin")
	_ = addCollaboratorCmd.MarkFlagRequired("repo")
	_ = addCollaboratorCmd.MarkFlagRequired("user")
	rootCmd.AddCommand(addCollaboratorCmd)
}
```

**Step 7: Update main.go**

```go
package main

import (
	"embed"

	"github.com/SammyLin/gh-ops/cmd"
)

//go:embed web/templates/* web/static/*
var templateFS embed.FS

func main() {
	cmd.SetTemplateFS(templateFS)
	cmd.Execute()
}
```

**Step 8: Verify it compiles**

Run: `cd /Users/sammylin/CascadeProjects/gh-ops && go build -o gh-ops .`
Expected: Compiles (commands registered but `runAction` not yet defined — will fail). That's OK, we define it in Task 2.

**Step 9: Commit**

```bash
git add cmd/root.go cmd/create_repo.go cmd/merge_pr.go cmd/create_tag.go cmd/add_collaborator.go main.go go.mod go.sum
git commit -m "feat: scaffold cobra CLI commands for all actions"
```

---

### Task 2: Implement local token cache

**Files:**
- Create: `internal/auth/token_store.go`
- Create: `internal/auth/token_store_test.go`

**Step 1: Write the test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/sammylin/CascadeProjects/gh-ops && go test -v ./internal/auth/`
Expected: FAIL — `NewTokenStore` and `CachedToken` not defined.

**Step 3: Write implementation**

```go
package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// CachedToken represents a locally cached OAuth token.
type CachedToken struct {
	AccessToken string    `json:"access_token"`
	GitHubUser  string    `json:"github_user"`
	SavedAt     time.Time `json:"saved_at"`
}

// TokenStore manages reading/writing OAuth tokens to a local file.
type TokenStore struct {
	path string
}

// NewTokenStore creates a TokenStore that persists to the given file path.
func NewTokenStore(path string) *TokenStore {
	return &TokenStore{path: path}
}

// DefaultTokenPath returns ~/.config/gh-ops/token.json.
func DefaultTokenPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	return filepath.Join(configDir, "gh-ops", "token.json")
}

// Save writes the token to disk with 0600 permissions.
func (s *TokenStore) Save(token *CachedToken) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

// Load reads the cached token from disk. Returns nil, nil if file does not exist.
func (s *TokenStore) Load() (*CachedToken, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var token CachedToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// Clear removes the cached token file.
func (s *TokenStore) Clear() error {
	err := os.Remove(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/sammylin/CascadeProjects/gh-ops && go test -v ./internal/auth/`
Expected: PASS (all 4 tests)

**Step 5: Commit**

```bash
git add internal/auth/token_store.go internal/auth/token_store_test.go
git commit -m "feat: add local token cache for OAuth tokens"
```

---

### Task 3: Implement ephemeral local server with confirm page and OAuth callback

**Files:**
- Create: `cmd/local_server.go`
- Create: `web/templates/confirm.html`

**Step 1: Create confirm.html template**

```html
{{template "base" .}}
{{define "content"}}
<div class="max-w-lg mx-auto mt-12">
  <div class="bg-white rounded-2xl shadow-sm border border-[#e8e4dd] p-8">
    <div class="text-center mb-6">
      <div class="w-16 h-16 bg-amber-100 rounded-full flex items-center justify-center mx-auto mb-4">
        <svg class="w-8 h-8 text-amber-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"/>
        </svg>
      </div>
      <h2 class="text-xl font-bold text-[#1a1a1a] mb-2">Confirm Action</h2>
    </div>

    <div class="bg-[#faf8f5] rounded-lg p-4 mb-6">
      <p class="text-sm text-[#434343] mb-2">Action</p>
      <p class="font-semibold text-[#1a1a1a]">{{.Action}}</p>

      {{if .Params}}
      <div class="mt-3 space-y-1">
        {{range $key, $value := .Params}}
        {{if $value}}
        <div class="flex justify-between text-sm">
          <span class="text-[#434343]">{{$key}}</span>
          <span class="text-[#1a1a1a] font-medium">{{$value}}</span>
        </div>
        {{end}}
        {{end}}
      </div>
      {{end}}
    </div>

    <a href="/authorize"
       class="block w-full text-center bg-[#ca8a04] hover:bg-[#a16207] text-white font-semibold py-3 px-6 rounded-lg transition-colors">
      Confirm & Authorize with GitHub
    </a>

    <p class="text-xs text-[#434343] text-center mt-4">
      You will be redirected to GitHub to authorize this operation.
    </p>
  </div>
</div>
{{end}}
```

**Step 2: Create cmd/local_server.go**

This is the core of the new architecture — the ephemeral local server + `runAction` function used by all CLI commands.

```go
package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/SammyLin/gh-ops/internal/actions"
	"github.com/SammyLin/gh-ops/internal/audit"
	"github.com/SammyLin/gh-ops/internal/auth"
	"github.com/SammyLin/gh-ops/internal/config"
	"golang.org/x/oauth2"
	githubOAuth "golang.org/x/oauth2/github"
)

func runAction(actionName string, params map[string]string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.IsActionAllowed(actionName) {
		return fmt.Errorf("action %q is not in the allowed list", actionName)
	}

	// Check for cached token
	tokenStore := auth.NewTokenStore(auth.DefaultTokenPath())
	cached, _ := tokenStore.Load()

	if cached != nil {
		// Try using cached token directly
		fmt.Printf("Using cached token for %s...\n", cached.GitHubUser)
		result, err := executeAction(cfg, actionName, params, cached.AccessToken, cached.GitHubUser)
		if err != nil {
			fmt.Printf("Cached token failed, re-authorizing...\n")
			_ = tokenStore.Clear()
		} else {
			fmt.Println(result)
			return nil
		}
	}

	// Start ephemeral local server for OAuth flow
	return runLocalServer(cfg, actionName, params, tokenStore)
}

func runLocalServer(cfg *config.Config, actionName string, params map[string]string, tokenStore *auth.TokenStore) error {
	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to find free port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	localBaseURL := fmt.Sprintf("http://localhost:%d", port)

	oauthCfg := &oauth2.Config{
		ClientID:     cfg.GitHub.ClientID,
		ClientSecret: cfg.GitHub.ClientSecret,
		Scopes:       []string{"repo"},
		Endpoint:     githubOAuth.Endpoint,
		RedirectURL:  localBaseURL + "/callback",
	}

	// Parse templates
	tmpl, err := template.ParseFS(templateFS, "web/templates/base.html", "web/templates/confirm.html", "web/templates/result.html", "web/templates/error.html")
	if err != nil {
		return fmt.Errorf("failed to parse templates: %w", err)
	}

	// Audit logger
	auditLogger, err := audit.New(cfg.Audit.DBPath)
	if err != nil {
		log.Printf("Warning: audit log disabled: %v", err)
	} else {
		defer auditLogger.Close()
	}

	// Channel to receive result
	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Generate OAuth state
	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	oauthState := hex.EncodeToString(stateBytes)

	mux := http.NewServeMux()

	// Confirm page
	mux.HandleFunc("GET /confirm", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "base", map[string]interface{}{
			"Action": actionName,
			"Params": params,
		})
	})

	// Start OAuth flow
	mux.HandleFunc("GET /authorize", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, oauthCfg.AuthCodeURL(oauthState), http.StatusFound)
	})

	// OAuth callback — exchange code, execute action, show result
	mux.HandleFunc("GET /callback", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("state") != oauthState {
			http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
			errCh <- fmt.Errorf("invalid OAuth state")
			return
		}

		token, err := oauthCfg.Exchange(req.Context(), req.URL.Query().Get("code"))
		if err != nil {
			renderErrorPage(tmpl, w, "OAuth exchange failed: "+err.Error(), http.StatusInternalServerError)
			errCh <- fmt.Errorf("OAuth exchange failed: %w", err)
			return
		}

		// Fetch GitHub username
		ghUser, err := auth.FetchGitHubUser(req.Context(), token.AccessToken)
		if err != nil {
			renderErrorPage(tmpl, w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
			errCh <- err
			return
		}

		// Cache token
		_ = tokenStore.Save(&auth.CachedToken{
			AccessToken: token.AccessToken,
			GitHubUser:  ghUser,
			SavedAt:     time.Now().UTC(),
		})

		// Execute the action
		result, err := executeAction(cfg, actionName, params, token.AccessToken, ghUser)

		// Audit log
		if auditLogger != nil {
			logResult := "success"
			if err != nil {
				logResult = "error: " + err.Error()
			}
			_ = auditLogger.Log(audit.Entry{
				GitHubUser: ghUser,
				Action:     actionName,
				Parameters: params,
				Result:     logResult,
				IPAddress:  "localhost",
			})
		}

		if err != nil {
			renderErrorPage(tmpl, w, err.Error(), http.StatusInternalServerError)
			errCh <- err
			return
		}

		tmpl.ExecuteTemplate(w, "base", map[string]interface{}{
			"Action":  actionName,
			"Message": result,
			"Success": true,
		})
		resultCh <- result
	})

	// Start server
	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	confirmURL := localBaseURL + "/confirm"
	fmt.Printf("\nOpen this link to authorize:\n\n    %s\n\nWaiting for authorization...\n", confirmURL)
	_ = openBrowser(confirmURL)

	// Wait for result or error
	select {
	case result := <-resultCh:
		fmt.Printf("\n✅ %s\n", result)
		_ = server.Shutdown(context.Background())
		return nil
	case err := <-errCh:
		_ = server.Shutdown(context.Background())
		return err
	}
}

func executeAction(cfg *config.Config, actionName string, params map[string]string, accessToken, ghUser string) (string, error) {
	return actions.Execute(actionName, params, accessToken)
}

func renderErrorPage(tmpl *template.Template, w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	tmpl.ExecuteTemplate(w, "base", map[string]interface{}{
		"Error":   true,
		"Message": message,
	})
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return nil
	}
}
```

**Step 3: Verify it compiles**

Run: `cd /Users/sammylin/CascadeProjects/gh-ops && go build -o gh-ops .`
Expected: Fails — `actions.Execute` and `auth.FetchGitHubUser` not yet defined. We add them in Task 4.

**Step 4: Commit**

```bash
git add cmd/local_server.go web/templates/confirm.html
git commit -m "feat: add ephemeral local server with confirm page and OAuth callback"
```

---

### Task 4: Refactor actions package to support direct execution

**Files:**
- Modify: `internal/actions/handler.go` — extract an `Execute` function
- Create: `internal/auth/github_user.go` — extract `FetchGitHubUser`

**Step 1: Create internal/auth/github_user.go**

```go
package auth

import (
	"context"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

// FetchGitHubUser returns the authenticated user's login name.
func FetchGitHubUser(ctx context.Context, accessToken string) (string, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	client := github.NewClient(oauth2.NewClient(ctx, ts))

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return "", err
	}
	return user.GetLogin(), nil
}
```

**Step 2: Extract `Execute` function from actions/handler.go**

Add a standalone `Execute` function at the bottom of `internal/actions/handler.go`:

```go
// Execute runs an action with the given parameters and access token.
// Returns a human-readable result message or an error.
func Execute(actionName string, params map[string]string, accessToken string) (string, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	client := github.NewClient(oauth2.NewClient(ctx, ts))

	switch actionName {
	case "create-repo":
		return executeCreateRepo(ctx, client, params)
	case "merge-pr":
		return executeMergePR(ctx, client, params)
	case "create-tag":
		return executeCreateTag(ctx, client, params)
	case "add-collaborator":
		return executeAddCollaborator(ctx, client, params)
	default:
		return "", fmt.Errorf("unknown action: %s", actionName)
	}
}
```

This requires refactoring the existing private methods (`createRepo`, `mergePR`, `createTag`, `addCollaborator`) on `Handler` into standalone functions (`executeCreateRepo`, `executeMergePR`, `executeCreateTag`, `executeAddCollaborator`) that take `(ctx, client, params)` instead of `(w, r)`.

The existing `HandleAction` method on `Handler` should be updated to call these extracted functions internally so the web-based flow still works.

Each extracted function signature:

```go
func executeCreateRepo(ctx context.Context, client *github.Client, params map[string]string) (string, error)
func executeMergePR(ctx context.Context, client *github.Client, params map[string]string) (string, error)
func executeCreateTag(ctx context.Context, client *github.Client, params map[string]string) (string, error)
func executeAddCollaborator(ctx context.Context, client *github.Client, params map[string]string) (string, error)
```

The logic in each function is identical to the current implementation but takes params from the map instead of `r.URL.Query()`, and returns `(string, error)` instead of writing to `http.ResponseWriter`.

**Step 3: Verify it compiles and tests pass**

Run: `cd /Users/sammylin/CascadeProjects/gh-ops && go build -o gh-ops . && go test -v ./...`
Expected: Compiles and all existing tests pass.

**Step 4: Commit**

```bash
git add internal/actions/handler.go internal/auth/github_user.go
git commit -m "refactor: extract Execute function and FetchGitHubUser for CLI usage"
```

---

### Task 5: Remove old long-running server mode and unused dependencies

**Files:**
- Delete: `cmd/server.go`
- Delete: `internal/auth/oauth.go` — entirely replaced by token_store.go + github_user.go
- Delete: `internal/middleware/ratelimit.go` — not needed for local ephemeral server
- Delete: `web/templates/home.html` (replaced by confirm.html)
- Delete: `web/templates/index.html` (unused legacy)

**Step 1: Delete old server.go**

Run: `rm /Users/sammylin/CascadeProjects/gh-ops/cmd/server.go`

**Step 2: Delete oauth.go entirely**

The cookie-based session auth is fully replaced by local token cache. All needed functions are in `token_store.go` and `github_user.go`.

Run: `rm /Users/sammylin/CascadeProjects/gh-ops/internal/auth/oauth.go`

**Step 3: Delete rate limiting middleware**

Run: `rm /Users/sammylin/CascadeProjects/gh-ops/internal/middleware/ratelimit.go`

**Step 4: Delete unused templates**

Run: `rm /Users/sammylin/CascadeProjects/gh-ops/web/templates/home.html /Users/sammylin/CascadeProjects/gh-ops/web/templates/index.html`

**Step 5: Remove chi, cors, gorilla/sessions dependencies**

Run: `cd /Users/sammylin/CascadeProjects/gh-ops && go mod tidy`

**Step 6: Verify it compiles and tests pass**

Run: `cd /Users/sammylin/CascadeProjects/gh-ops && go build -o gh-ops . && go test -v ./...`
Expected: Compiles, tests pass. chi, cors, gorilla/sessions should be gone from go.sum.

**Step 7: Commit**

```bash
git add -A
git commit -m "refactor: remove old server mode, chi, cors, gorilla/sessions"
```

---

### Task 6: Add `logout` command to clear cached token

**Files:**
- Create: `cmd/logout.go`

**Step 1: Create cmd/logout.go**

```go
package cmd

import (
	"fmt"

	"github.com/SammyLin/gh-ops/internal/auth"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear cached GitHub OAuth token",
	RunE: func(cmd *cobra.Command, args []string) error {
		store := auth.NewTokenStore(auth.DefaultTokenPath())
		if err := store.Clear(); err != nil {
			return fmt.Errorf("failed to clear token: %w", err)
		}
		fmt.Println("Logged out. Cached token removed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/sammylin/CascadeProjects/gh-ops && go build -o gh-ops .`
Expected: Compiles.

**Step 3: Commit**

```bash
git add cmd/logout.go
git commit -m "feat: add logout command to clear cached token"
```

---

### Task 7: Update config — remove session secret, simplify

**Files:**
- Modify: `internal/config/config.go` — remove `SessionConfig`
- Modify: `internal/config/config_test.go` — update tests
- Modify: `config.yaml` — remove session section
- Modify: `.env.example` — remove `SESSION_SECRET`

**Step 1: Update config.go**

Remove `SessionConfig` struct and `Session` field from `Config`. The `SESSION_SECRET` is no longer needed since we use local file token storage instead of encrypted cookies.

```go
type Config struct {
	Server         ServerConfig `yaml:"server"`
	GitHub         GitHubConfig `yaml:"github"`
	AllowedActions []string     `yaml:"allowed_actions"`
	Audit          AuditConfig  `yaml:"audit"`
}
```

**Step 2: Update config.yaml**

Remove `session:` block entirely.

**Step 3: Update .env.example**

Remove `SESSION_SECRET` line.

**Step 4: Update config_test.go**

Remove any assertions about `Session.Secret`.

**Step 5: Run tests**

Run: `cd /Users/sammylin/CascadeProjects/gh-ops && go test -v ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go config.yaml .env.example
git commit -m "chore: remove session config (tokens cached locally now)"
```

---

### Task 8: Update README and SPEC

**Files:**
- Modify: `README.md`
- Modify: `SPEC.md`

**Step 1: Update README.md**

Key changes:
- Update description from "web service" to "CLI tool"
- Replace installation/usage to show CLI commands
- Remove `SESSION_SECRET` references
- Update directory structure (remove server.go, home.html, middleware/)
- Remove web server API reference (no more `/health`, `/auth/login` etc.)
- Remove rate limiting from security section
- Update development section

**Step 2: Update SPEC.md**

Key changes:
- Update overview from "web service" to "CLI tool"
- Update flow description
- Remove session/CORS/rate-limiting from tech stack
- Update directory structure

**Step 3: Commit**

```bash
git add README.md SPEC.md
git commit -m "docs: update README and SPEC for CLI-based flow"
```

---

### Task 9: Update CI workflows and goreleaser

**Files:**
- Modify: `.github/workflows/build.yml` — verify it still works (just Go build)
- Modify: `.goreleaser.yml` — remove `config.yaml` from extra files (optional, keep if still useful)

**Step 1: Verify build workflow**

The build workflow should already work (just `go build`). Verify no references to removed files.

**Step 2: Run full build**

Run: `cd /Users/sammylin/CascadeProjects/gh-ops && go build -o gh-ops . && go test -v ./... && echo "ALL GOOD"`

**Step 3: Test CLI commands work**

Run:
```bash
./gh-ops --help
./gh-ops create-repo --help
./gh-ops merge-pr --help
./gh-ops create-tag --help
./gh-ops add-collaborator --help
./gh-ops logout --help
```

**Step 4: Commit any remaining changes**

```bash
git add -A
git commit -m "chore: finalize CLI migration, update CI config"
```

---

## Summary of Architecture Changes

| Before (Web Server) | After (CLI) |
|---|---|
| Long-running HTTP server | Ephemeral localhost server per command |
| Cookie-based sessions (gorilla/sessions) | Local file token cache (`~/.config/gh-ops/token.json`) |
| `SESSION_SECRET` env var required | No session secret needed |
| Rate limiting middleware | Removed (localhost only) |
| CORS middleware | Removed (no cross-origin) |
| chi router + cors middleware | net/http stdlib |
| Home page (`/`) | Confirm page in browser |
| `flag`-based `--config` | cobra CLI with subcommands |
| Agent generates URL for user to click | User runs CLI command directly |

## Files Created
- `cmd/root.go` — cobra root command
- `cmd/create_repo.go` — create-repo subcommand
- `cmd/merge_pr.go` — merge-pr subcommand
- `cmd/create_tag.go` — create-tag subcommand
- `cmd/add_collaborator.go` — add-collaborator subcommand
- `cmd/logout.go` — logout subcommand
- `cmd/local_server.go` — ephemeral server + `runAction` logic
- `internal/auth/token_store.go` — local token cache
- `internal/auth/token_store_test.go` — token cache tests
- `internal/auth/github_user.go` — GitHub user fetch helper
- `web/templates/confirm.html` — action confirmation page

## Files Deleted
- `cmd/server.go` — old long-running server
- `internal/auth/oauth.go` — cookie-based OAuth (replaced by token_store.go + github_user.go)
- `internal/middleware/ratelimit.go` — rate limiting
- `web/templates/home.html` — old landing page
- `web/templates/index.html` — unused legacy

## Dependencies Removed
- `github.com/go-chi/chi/v5` — replaced by net/http stdlib
- `github.com/go-chi/cors` — not needed for localhost
- `github.com/gorilla/sessions` — replaced by local file token cache

## Files Modified
- `main.go` — switch to cobra
- `internal/actions/handler.go` — extract `Execute()` function
- `internal/config/config.go` — remove `SessionConfig`
- `internal/config/config_test.go` — update tests
- `config.yaml` — remove session section
- `.env.example` — remove `SESSION_SECRET`
- `README.md` — full rewrite for CLI usage
- `SPEC.md` — update for CLI architecture
