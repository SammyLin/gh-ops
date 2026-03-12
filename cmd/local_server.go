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

	return runLocalServer(cfg, actionName, params, tokenStore)
}

func runLocalServer(cfg *config.Config, actionName string, params map[string]string, tokenStore *auth.TokenStore) error {
	if cfg.GitHub.ClientID == "" || cfg.GitHub.ClientSecret == "" {
		return fmt.Errorf("GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET must be set (check your environment or config file)")
	}

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

	tmpl, err := template.ParseFS(templateFS, "web/templates/base.html", "web/templates/confirm.html", "web/templates/result.html", "web/templates/error.html")
	if err != nil {
		return fmt.Errorf("failed to parse templates: %w", err)
	}

	auditLogger, err := audit.New(cfg.Audit.DBPath)
	if err != nil {
		log.Printf("Warning: audit log disabled: %v", err)
	} else {
		defer auditLogger.Close()
	}

	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)

	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	oauthState := hex.EncodeToString(stateBytes)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /confirm", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "base", map[string]interface{}{
			"Action": actionName,
			"Params": params,
		})
	})

	mux.HandleFunc("GET /authorize", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, oauthCfg.AuthCodeURL(oauthState), http.StatusFound)
	})

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

		ghUser, err := auth.FetchGitHubUser(req.Context(), token.AccessToken)
		if err != nil {
			renderErrorPage(tmpl, w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
			errCh <- err
			return
		}

		_ = tokenStore.Save(&auth.CachedToken{
			AccessToken: token.AccessToken,
			GitHubUser:  ghUser,
			SavedAt:     time.Now().UTC(),
		})

		result, err := executeAction(cfg, actionName, params, token.AccessToken, ghUser)

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

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	confirmURL := localBaseURL + "/confirm"
	fmt.Printf("\nOpen this link to authorize:\n\n    %s\n\nWaiting for authorization...\n", confirmURL)
	_ = openBrowser(confirmURL)

	select {
	case result := <-resultCh:
		fmt.Printf("\n%s %s\n", "\u2705", result)
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
