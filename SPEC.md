# gh-ops — GitHub Operations CLI

## Overview
A lightweight Go CLI tool that provides one-click GitHub operations via OAuth.
Agent suggests a CLI command -> user runs it -> ephemeral server starts -> browser opens confirm page -> GitHub OAuth -> action executed -> server shuts down.

## Flow
1. Agent suggests command: `gh-ops create-repo --name homebrew-tap --visibility public`
2. User runs the command
3. Ephemeral localhost server starts, browser opens confirmation page
4. Redirects to GitHub OAuth consent (first time only, token cached at `~/.config/gh-ops/token.json`)
5. Action executes using user's authorization
6. Result displayed in browser, server shuts down

## Supported Actions (MVP)

### create-repo
- `--name` (required) — repo name
- `--visibility` — public/private (default: public)
- `--description` — repo description
- `--auto-init` — create with README (default: true)

### merge-pr
- `--repo` (required) — owner/repo
- `--pr-number` (required) — PR number
- `--merge-method` — merge/squash/rebase (default: merge)

### create-tag
- `--repo` (required) — owner/repo
- `--tag` (required) — tag name (e.g., v1.0.0)
- `--sha` — commit SHA (default: HEAD of default branch)
- `--message` — tag message

### add-collaborator
- `--repo` (required) — owner/repo
- `--user` (required) — GitHub username
- `--permission` — pull/push/admin (default: push)

### logout
- Removes cached OAuth token from `~/.config/gh-ops/token.json`

## Technical Stack
- Go + Cobra CLI framework
- GitHub OAuth App (ephemeral localhost server for OAuth flow)
- SQLite for audit log
- Token cached locally at `~/.config/gh-ops/token.json`

## GitHub OAuth Setup
- Create OAuth App at https://github.com/settings/developers
- Callback URL: `http://localhost:9091/auth/callback`
- Scopes needed: `repo`, `admin:org` (for creating repos)

## Audit Log
Every action logged to SQLite:
- timestamp
- github_user
- action
- parameters
- result (success/error)

## Security
- OAuth tokens cached locally at `~/.config/gh-ops/token.json`
- Ephemeral server — localhost server runs only during the OAuth/action flow
- CSRF protection via OAuth state parameter
- Action allowlist configurable via config.yaml

## Config (config.yaml)
```yaml
server:
  port: 9091
  base_url: http://localhost:9091

github:
  client_id: xxx
  client_secret: xxx

allowed_actions:
  - create-repo
  - merge-pr
  - create-tag
  - add-collaborator

audit:
  db_path: ./audit.db
```

## Directory Structure
```
gh-ops/
├── main.go                      # Entry point, embed templates
├── cmd/
│   ├── root.go                  # Cobra root command
│   ├── create_repo.go           # create-repo subcommand
│   ├── merge_pr.go              # merge-pr subcommand
│   ├── create_tag.go            # create-tag subcommand
│   ├── add_collaborator.go      # add-collaborator subcommand
│   ├── logout.go                # logout subcommand
│   └── local_server.go          # Ephemeral OAuth server
├── internal/
│   ├── actions/                 # GitHub operations
│   ├── auth/                    # Token cache + GitHub user fetch
│   ├── audit/                   # SQLite audit log
│   └── config/                  # Config loading
├── web/
│   └── templates/               # HTML confirm/result/error pages
├── config.yaml
├── go.mod
└── README.md
```
