package actions

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

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
