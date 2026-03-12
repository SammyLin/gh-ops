package auth

import (
	"context"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

func FetchGitHubUser(ctx context.Context, accessToken string) (string, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	client := github.NewClient(oauth2.NewClient(ctx, ts))

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return "", err
	}
	return user.GetLogin(), nil
}
