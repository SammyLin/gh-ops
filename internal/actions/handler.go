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

	var (
		result string
		params map[string]string
		err    error
	)

	switch action {
	case "create-repo":
		result, params, err = h.createRepo(r.Context(), client, r)
	case "merge-pr":
		result, params, err = h.mergePR(r.Context(), client, r)
	case "create-tag":
		result, params, err = h.createTag(r.Context(), client, r)
	case "add-collaborator":
		result, params, err = h.addCollaborator(r.Context(), client, r)
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

func (h *Handler) createRepo(ctx context.Context, client *github.Client, r *http.Request) (string, map[string]string, error) {
	name := r.URL.Query().Get("name")
	if name == "" {
		return "", nil, fmt.Errorf("name is required")
	}

	visibility := r.URL.Query().Get("visibility")
	if visibility == "" {
		visibility = "public"
	}
	description := r.URL.Query().Get("description")
	autoInit := r.URL.Query().Get("auto_init") != "false"

	params := map[string]string{
		"name":       name,
		"visibility": visibility,
	}

	repo, _, err := client.Repositories.Create(ctx, "", &github.Repository{
		Name:        github.String(name),
		Description: github.String(description),
		Private:     github.Bool(visibility == "private"),
		AutoInit:    github.Bool(autoInit),
	})
	if err != nil {
		return "", params, fmt.Errorf("failed to create repo: %w", err)
	}

	return fmt.Sprintf("%s created", repo.GetFullName()), params, nil
}

func (h *Handler) mergePR(ctx context.Context, client *github.Client, r *http.Request) (string, map[string]string, error) {
	repoFullName := r.URL.Query().Get("repo")
	prNumberStr := r.URL.Query().Get("pr_number")
	mergeMethod := r.URL.Query().Get("merge_method")

	if repoFullName == "" || prNumberStr == "" {
		return "", nil, fmt.Errorf("repo and pr_number are required")
	}

	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("repo must be in owner/repo format")
	}
	owner, repo := parts[0], parts[1]

	prNumber, err := strconv.Atoi(prNumberStr)
	if err != nil {
		return "", nil, fmt.Errorf("invalid pr_number: %w", err)
	}

	if mergeMethod == "" {
		mergeMethod = "merge"
	}

	params := map[string]string{
		"repo":         repoFullName,
		"pr_number":    prNumberStr,
		"merge_method": mergeMethod,
	}

	mergeResult, _, err := client.PullRequests.Merge(ctx, owner, repo, prNumber, "", &github.PullRequestOptions{
		MergeMethod: mergeMethod,
	})
	if err != nil {
		return "", params, fmt.Errorf("failed to merge PR: %w", err)
	}

	return fmt.Sprintf("PR #%d merged in %s (%s)", prNumber, repoFullName, mergeResult.GetMessage()), params, nil
}

func (h *Handler) createTag(ctx context.Context, client *github.Client, r *http.Request) (string, map[string]string, error) {
	repoFullName := r.URL.Query().Get("repo")
	tag := r.URL.Query().Get("tag")
	sha := r.URL.Query().Get("sha")
	message := r.URL.Query().Get("message")

	if repoFullName == "" || tag == "" {
		return "", nil, fmt.Errorf("repo and tag are required")
	}

	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("repo must be in owner/repo format")
	}
	owner, repo := parts[0], parts[1]

	params := map[string]string{
		"repo": repoFullName,
		"tag":  tag,
		"sha":  sha,
	}

	// If no SHA provided, resolve HEAD of default branch
	if sha == "" {
		repoInfo, _, err := client.Repositories.Get(ctx, owner, repo)
		if err != nil {
			return "", params, fmt.Errorf("failed to get repo info: %w", err)
		}
		ref, _, err := client.Git.GetRef(ctx, owner, repo, "heads/"+repoInfo.GetDefaultBranch())
		if err != nil {
			return "", params, fmt.Errorf("failed to get default branch HEAD: %w", err)
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
			return "", params, fmt.Errorf("failed to create tag object: %w", err)
		}
		_, _, err = client.Git.CreateRef(ctx, owner, repo, &github.Reference{
			Ref:    github.String("refs/tags/" + tag),
			Object: &github.GitObject{SHA: tagObj.SHA},
		})
		if err != nil {
			return "", params, fmt.Errorf("failed to create tag ref: %w", err)
		}
	} else {
		// Lightweight tag
		_, _, err := client.Git.CreateRef(ctx, owner, repo, &github.Reference{
			Ref:    github.String("refs/tags/" + tag),
			Object: &github.GitObject{SHA: github.String(sha)},
		})
		if err != nil {
			return "", params, fmt.Errorf("failed to create tag: %w", err)
		}
	}

	shortSHA := sha
	if len(sha) > 8 {
		shortSHA = sha[:8]
	}
	return fmt.Sprintf("Tag %s created in %s (at %s)", tag, repoFullName, shortSHA), params, nil
}

func (h *Handler) addCollaborator(ctx context.Context, client *github.Client, r *http.Request) (string, map[string]string, error) {
	repo := r.URL.Query().Get("repo")
	user := r.URL.Query().Get("user")
	permission := r.URL.Query().Get("permission")

	if repo == "" || user == "" {
		return "", nil, fmt.Errorf("repo and user are required")
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
			return "", nil, fmt.Errorf("failed to get authenticated user: %w", err)
		}
		owner = authUser.GetLogin()
	}

	params := map[string]string{
		"repo":       owner + "/" + repoName,
		"user":       user,
		"permission": permission,
	}

	_, _, err := client.Repositories.AddCollaborator(ctx, owner, repoName, user, &github.RepositoryAddCollaboratorOptions{
		Permission: permission,
	})
	if err != nil {
		return "", params, fmt.Errorf("failed to add collaborator: %w", err)
	}

	return fmt.Sprintf("%s added to %s/%s with %s permission", user, owner, repoName, permission), params, nil
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
