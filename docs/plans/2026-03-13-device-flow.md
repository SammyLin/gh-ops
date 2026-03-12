# Device Flow Authentication Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the local server OAuth flow with GitHub Device Flow so the CLI doesn't need a redirect URI or local HTTP server.

**Architecture:** New `internal/auth/device_flow.go` handles the Device Flow HTTP calls (request user code, poll for token). `cmd/local_server.go` is replaced with `cmd/device_auth.go` that shows action confirmation in the terminal, runs Device Flow, then executes the action. Templates and local server code are removed.

**Tech Stack:** Go, GitHub Device Flow API (REST), existing token cache and audit logging.

---

### Task 1: Create Device Flow auth module

**Files:**
- Create: `internal/auth/device_flow.go`
- Test: `internal/auth/device_flow_test.go`

**Step 1: Write the failing test**

```go
// internal/auth/device_flow_test.go
package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestDeviceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.FormValue("client_id") != "test-client-id" {
			t.Fatalf("expected client_id=test-client-id, got %s", r.FormValue("client_id"))
		}
		if r.FormValue("scope") != "repo" {
			t.Fatalf("expected scope=repo, got %s", r.FormValue("scope"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DeviceCodeResponse{
			DeviceCode:      "device-123",
			UserCode:        "ABCD-1234",
			VerificationURI: "https://github.com/login/device",
			ExpiresIn:       900,
			Interval:        5,
		})
	}))
	defer server.Close()

	resp, err := RequestDeviceCode(server.URL, "test-client-id", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UserCode != "ABCD-1234" {
		t.Fatalf("expected user code ABCD-1234, got %s", resp.UserCode)
	}
	if resp.DeviceCode != "device-123" {
		t.Fatalf("expected device code device-123, got %s", resp.DeviceCode)
	}
}

func TestPollForToken_Success(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount < 2 {
			json.NewEncoder(w).Encode(tokenPollResponse{
				Error: "authorization_pending",
			})
			return
		}
		json.NewEncoder(w).Encode(tokenPollResponse{
			AccessToken: "gho_test_token_123",
			TokenType:   "bearer",
		})
	}))
	defer server.Close()

	token, err := PollForToken(server.URL, "test-client-id", "device-123", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "gho_test_token_123" {
		t.Fatalf("expected gho_test_token_123, got %s", token)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/auth/ -run TestRequestDeviceCode -v`
Expected: FAIL - types/functions not defined

**Step 3: Write minimal implementation**

```go
// internal/auth/device_flow.go
package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	GitHubDeviceCodeURL = "https://github.com/login/device/code"
	GitHubTokenURL      = "https://github.com/login/oauth/access_token"
)

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenPollResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
}

func RequestDeviceCode(endpoint, clientID, scope string) (*DeviceCodeResponse, error) {
	resp, err := http.PostForm(endpoint, url.Values{
		"client_id": {clientID},
		"scope":     {scope},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to request device code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request failed with status %d", resp.StatusCode)
	}

	var result DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode device code response: %w", err)
	}
	return &result, nil
}

func PollForToken(endpoint, clientID, deviceCode string, interval int) (string, error) {
	if interval < 1 {
		interval = 5
	}

	for {
		time.Sleep(time.Duration(interval) * time.Second)

		resp, err := http.PostForm(endpoint, url.Values{
			"client_id":   {clientID},
			"device_code": {deviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		})
		if err != nil {
			return "", fmt.Errorf("failed to poll for token: %w", err)
		}

		var result tokenPollResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return "", fmt.Errorf("failed to decode token response: %w", err)
		}
		resp.Body.Close()

		switch result.Error {
		case "":
			return result.AccessToken, nil
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5
			continue
		case "expired_token":
			return "", fmt.Errorf("device code expired, please try again")
		case "access_denied":
			return "", fmt.Errorf("authorization denied by user")
		default:
			return "", fmt.Errorf("unexpected error: %s", result.Error)
		}
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/auth/ -run "TestRequestDeviceCode|TestPollForToken" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/auth/device_flow.go internal/auth/device_flow_test.go
git commit -m "feat: add Device Flow auth module"
```

---

### Task 2: Replace local server with Device Flow in CLI

**Files:**
- Create: `cmd/device_auth.go`
- Modify: `cmd/local_server.go` (delete `runLocalServer`, keep `runAction` and `executeAction`)

**Step 1: Create device_auth.go**

```go
// cmd/device_auth.go
package cmd

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"time"

	"github.com/SammyLin/gh-ops/internal/actions"
	"github.com/SammyLin/gh-ops/internal/audit"
	"github.com/SammyLin/gh-ops/internal/auth"
	"github.com/SammyLin/gh-ops/internal/config"
)

func runDeviceFlow(cfg *config.Config, actionName string, params map[string]string, tokenStore *auth.TokenStore) error {
	if cfg.GitHub.ClientID == "" {
		return fmt.Errorf("GITHUB_CLIENT_ID must be set (check your environment or config file)")
	}

	// Show action confirmation in terminal
	fmt.Printf("\n  Action:  %s\n", actionName)
	for k, v := range params {
		if v != "" {
			fmt.Printf("  %-8s %s\n", k+":", v)
		}
	}
	fmt.Println()

	// Request device code
	deviceResp, err := auth.RequestDeviceCode(auth.GitHubDeviceCodeURL, cfg.GitHub.ClientID, "repo")
	if err != nil {
		return err
	}

	// Show user code and open browser
	fmt.Printf("  Please open:  %s\n", deviceResp.VerificationURI)
	fmt.Printf("  And enter:    %s\n\n", deviceResp.UserCode)
	fmt.Printf("Waiting for authorization...\n")
	_ = openBrowser(deviceResp.VerificationURI)

	// Poll for token
	accessToken, err := auth.PollForToken(auth.GitHubTokenURL, cfg.GitHub.ClientID, deviceResp.DeviceCode, deviceResp.Interval)
	if err != nil {
		return err
	}

	// Fetch GitHub user
	ghUser, err := auth.FetchGitHubUser(context.Background(), accessToken)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// Cache token
	_ = tokenStore.Save(&auth.CachedToken{
		AccessToken: accessToken,
		GitHubUser:  ghUser,
		SavedAt:     time.Now().UTC(),
	})

	// Execute action
	result, err := actions.Execute(actionName, params, accessToken)

	// Audit log
	auditLogger, auditErr := audit.New(cfg.Audit.DBPath)
	if auditErr != nil {
		log.Printf("Warning: audit log disabled: %v", auditErr)
	} else {
		defer auditLogger.Close()
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
		return err
	}

	fmt.Printf("\n✅ %s\n", result)
	return nil
}
```

**Step 2: Update runAction to call runDeviceFlow instead of runLocalServer**

In `cmd/local_server.go`, change line 50:
```go
// Before:
return runLocalServer(cfg, actionName, params, tokenStore)
// After:
return runDeviceFlow(cfg, actionName, params, tokenStore)
```

Then delete `runLocalServer`, `renderErrorPage`, and remove unused imports (`html/template`, `net`, `net/http`, `crypto/rand`, `encoding/hex`, `golang.org/x/oauth2`, `golang.org/x/oauth2/github`).

Rename `local_server.go` to `run_action.go` since it no longer runs a local server.

**Step 3: Clean up config — client_secret no longer required**

In `config.yaml`, remove the `client_secret` line. In `config.go`, remove `ClientSecret` from `GitHubConfig`.

**Step 4: Clean up main.go — templateFS no longer needed for CLI**

Remove `templateFS` embed and `SetTemplateFS` call from `main.go`. Remove `templateFS` variable and `SetTemplateFS` from `cmd/create_repo.go`.

**Step 5: Build and verify**

Run: `go build -o gh-ops .`
Expected: builds successfully

**Step 6: Commit**

```bash
git add -A
git commit -m "feat: replace local server OAuth with GitHub Device Flow"
```

---

### Task 3: Clean up unused files

**Files:**
- Delete: `web/templates/confirm.html`
- Delete: `web/templates/error.html`
- Delete: `web/templates/result.html`
- Delete: `web/templates/base.html`
- Keep: `web/static/*` (if used elsewhere, otherwise delete)

**Step 1: Remove unused template files and web embed**

Delete all template files that were only used by the local server flow.

**Step 2: Update .env.example**

Remove `GITHUB_CLIENT_SECRET` from `.env.example` since Device Flow doesn't need it.

**Step 3: Build and verify**

Run: `go build -o gh-ops .`
Expected: builds successfully

**Step 4: Manual test**

Run: `./gh-ops create-repo --name test-device-flow`
Expected:
```
  Action:  create-repo
  name:    test-device-flow

  Please open:  https://github.com/login/device
  And enter:    XXXX-XXXX

Waiting for authorization...
```

**Step 5: Commit**

```bash
git add -A
git commit -m "chore: remove unused local server templates and client_secret config"
```

---

### Notes

- GitHub OAuth App 需要在 Settings > Developer settings > OAuth Apps 裡勾選 **"Enable Device Flow"**
- Device Flow 只需要 `client_id`，不需要 `client_secret`
- `openBrowser()` function 保留在 `device_auth.go` 或獨立檔案中
