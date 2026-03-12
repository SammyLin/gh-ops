package actions

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"

	"github.com/SammyLin/gh-ops/internal/audit"
	"github.com/SammyLin/gh-ops/internal/auth"
)

// Handler executes GitHub operations and renders result pages.
type Handler struct {
	audit   *audit.Logger
	tmpl    *template.Template
	allowed map[string]bool
}

// NewHandler creates an action handler with the given audit logger, templates, and allowlist.
func NewHandler(auditLogger *audit.Logger, tmpl *template.Template, allowedActions []string) *Handler {
	allowed := make(map[string]bool, len(allowedActions))
	for _, a := range allowedActions {
		allowed[a] = true
	}
	return &Handler{audit: auditLogger, tmpl: tmpl, allowed: allowed}
}

// Execute runs an action by name with the given parameters and access token.
// It creates a GitHub client from the token and dispatches to the appropriate function.
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

// HandleAction routes to the appropriate GitHub operation based on the URL path.
func (h *Handler) HandleAction(w http.ResponseWriter, r *http.Request) {
	action := chi.URLParam(r, "action")

	if !h.allowed[action] {
		h.renderError(w, "Action not allowed: "+action, http.StatusForbidden)
		return
	}

	token := auth.TokenFromContext(r.Context())
	user := auth.UserFromContext(r.Context())

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := github.NewClient(oauth2.NewClient(r.Context(), ts))

	// Collect query parameters into a map
	params := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	var (
		result string
		err    error
	)

	switch action {
	case "create-repo":
		result, err = executeCreateRepo(r.Context(), client, params)
	case "merge-pr":
		result, err = executeMergePR(r.Context(), client, params)
	case "create-tag":
		result, err = executeCreateTag(r.Context(), client, params)
	case "add-collaborator":
		result, err = executeAddCollaborator(r.Context(), client, params)
	default:
		h.renderError(w, "Unknown action: "+action, http.StatusBadRequest)
		return
	}

	logResult := "success"
	if err != nil {
		logResult = "error: " + err.Error()
	}

	_ = h.audit.Log(audit.Entry{
		GitHubUser: user,
		Action:     action,
		Parameters: params,
		Result:     logResult,
		IPAddress:  realIP(r),
	})

	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.renderResult(w, action, result)
}

func executeCreateRepo(ctx context.Context, client *github.Client, params map[string]string) (string, error) {
	name := params["name"]
	if name == "" {
		return "", fmt.Errorf("name is required")
	}

	visibility := params["visibility"]
	if visibility == "" {
		visibility = "public"
	}
	description := params["description"]
	autoInit := params["auto_init"] != "false"

	repo, _, err := client.Repositories.Create(ctx, "", &github.Repository{
		Name:        github.String(name),
		Description: github.String(description),
		Private:     github.Bool(visibility == "private"),
		AutoInit:    github.Bool(autoInit),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create repo: %w", err)
	}

	return fmt.Sprintf("%s created", repo.GetFullName()), nil
}

func executeMergePR(ctx context.Context, client *github.Client, params map[string]string) (string, error) {
	repoFullName := params["repo"]
	prNumberStr := params["pr_number"]
	mergeMethod := params["merge_method"]

	if repoFullName == "" || prNumberStr == "" {
		return "", fmt.Errorf("repo and pr_number are required")
	}

	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("repo must be in owner/repo format")
	}
	owner, repo := parts[0], parts[1]

	prNumber, err := strconv.Atoi(prNumberStr)
	if err != nil {
		return "", fmt.Errorf("invalid pr_number: %w", err)
	}

	if mergeMethod == "" {
		mergeMethod = "merge"
	}

	mergeResult, _, err := client.PullRequests.Merge(ctx, owner, repo, prNumber, "", &github.PullRequestOptions{
		MergeMethod: mergeMethod,
	})
	if err != nil {
		return "", fmt.Errorf("failed to merge PR: %w", err)
	}

	return fmt.Sprintf("PR #%d merged in %s (%s)", prNumber, repoFullName, mergeResult.GetMessage()), nil
}

func executeCreateTag(ctx context.Context, client *github.Client, params map[string]string) (string, error) {
	repoFullName := params["repo"]
	tag := params["tag"]
	sha := params["sha"]
	message := params["message"]

	if repoFullName == "" || tag == "" {
		return "", fmt.Errorf("repo and tag are required")
	}

	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("repo must be in owner/repo format")
	}
	owner, repo := parts[0], parts[1]

	// If no SHA provided, resolve HEAD of default branch
	if sha == "" {
		repoInfo, _, err := client.Repositories.Get(ctx, owner, repo)
		if err != nil {
			return "", fmt.Errorf("failed to get repo info: %w", err)
		}
		ref, _, err := client.Git.GetRef(ctx, owner, repo, "heads/"+repoInfo.GetDefaultBranch())
		if err != nil {
			return "", fmt.Errorf("failed to get default branch HEAD: %w", err)
		}
		sha = ref.GetObject().GetSHA()
	}

	if message != "" {
		// Annotated tag
		tagObj, _, err := client.Git.CreateTag(ctx, owner, repo, &github.Tag{
			Tag:     github.String(tag),
			Message: github.String(message),
			Object:  &github.GitObject{SHA: github.String(sha), Type: github.String("commit")},
		})
		if err != nil {
			return "", fmt.Errorf("failed to create tag object: %w", err)
		}
		_, _, err = client.Git.CreateRef(ctx, owner, repo, &github.Reference{
			Ref:    github.String("refs/tags/" + tag),
			Object: &github.GitObject{SHA: tagObj.SHA},
		})
		if err != nil {
			return "", fmt.Errorf("failed to create tag ref: %w", err)
		}
	} else {
		// Lightweight tag
		_, _, err := client.Git.CreateRef(ctx, owner, repo, &github.Reference{
			Ref:    github.String("refs/tags/" + tag),
			Object: &github.GitObject{SHA: github.String(sha)},
		})
		if err != nil {
			return "", fmt.Errorf("failed to create tag: %w", err)
		}
	}

	shortSHA := sha
	if len(sha) > 8 {
		shortSHA = sha[:8]
	}
	return fmt.Sprintf("Tag %s created in %s (at %s)", tag, repoFullName, shortSHA), nil
}

func executeAddCollaborator(ctx context.Context, client *github.Client, params map[string]string) (string, error) {
	repo := params["repo"]
	user := params["user"]
	permission := params["permission"]

	if repo == "" || user == "" {
		return "", fmt.Errorf("repo and user are required")
	}

	if permission == "" {
		permission = "push"
	}

	// repo can be "owner/repo" or just "repo" (defaults to authenticated user)
	var owner, repoName string
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) == 2 {
		owner, repoName = parts[0], parts[1]
	} else {
		repoName = repo
		// Get authenticated user as owner
		authUser, _, err := client.Users.Get(ctx, "")
		if err != nil {
			return "", fmt.Errorf("failed to get authenticated user: %w", err)
		}
		owner = authUser.GetLogin()
	}

	_, _, err := client.Repositories.AddCollaborator(ctx, owner, repoName, user, &github.RepositoryAddCollaboratorOptions{
		Permission: permission,
	})
	if err != nil {
		return "", fmt.Errorf("failed to add collaborator: %w", err)
	}

	return fmt.Sprintf("%s added to %s/%s with %s permission", user, owner, repoName, permission), nil
}

func (h *Handler) renderResult(w http.ResponseWriter, action, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "result.html", map[string]string{
		"Action":  action,
		"Message": message,
	})
}

func (h *Handler) renderError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = h.tmpl.ExecuteTemplate(w, "error.html", map[string]string{
		"Message": message,
	})
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
