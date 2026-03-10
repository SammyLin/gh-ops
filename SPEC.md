# gh-ops — GitHub Operations Web API

## Overview
A lightweight Go web service that provides one-click GitHub operations via OAuth.
Agent/Mochi generates a URL → Sammy clicks → GitHub OAuth → action executed → done.

## Flow
1. Agent generates link: `https://ghops.3mi.tw/action/create-repo?name=homebrew-tap&visibility=public`
2. Sammy clicks the link
3. Redirects to GitHub OAuth consent (first time only, token cached after)
4. Action executes using Sammy's authorization
5. Shows result page: "✅ SammyLin/homebrew-tap created"

## Supported Actions (MVP)

### POST /api/create-repo
- `name` (required) — repo name
- `visibility` — public/private (default: public)
- `description` — repo description
- `auto_init` — create with README (default: true)

### POST /api/merge-pr
- `repo` (required) — owner/repo
- `pr_number` (required) — PR number
- `merge_method` — merge/squash/rebase (default: merge)

### POST /api/create-tag
- `repo` (required) — owner/repo
- `tag` (required) — tag name (e.g., v1.0.0)
- `sha` — commit SHA (default: HEAD of default branch)
- `message` — tag message

## Technical Stack
- Go + chi router
- GitHub OAuth App (OAuth flow for user authorization)
- SQLite for audit log
- Docker packaged
- CORS enabled for API calls

## GitHub OAuth Setup
- Create OAuth App at https://github.com/settings/developers
- Callback URL: `https://ghops.3mi.tw/auth/callback`
- Scopes needed: `repo`, `admin:org` (for creating repos)

## Audit Log
Every action logged to SQLite:
- timestamp
- github_user
- action
- parameters
- result (success/error)
- ip_address

## Security
- OAuth tokens stored encrypted (age via psw-cli pattern)
- HTTPS only (Cloudflare Tunnel)
- Rate limiting
- Action allowlist configurable via config.yaml

## Config (config.yaml)
```yaml
server:
  port: 8080
  base_url: https://ghops.3mi.tw

github:
  client_id: xxx
  client_secret: xxx  # or from psw-cli

allowed_actions:
  - create-repo
  - merge-pr
  - create-tag

audit:
  db_path: ./audit.db
```

## Directory Structure
```
gh-ops/
├── main.go
├── cmd/
│   └── server.go
├── internal/
│   ├── auth/        # GitHub OAuth flow
│   ├── actions/     # GitHub operations
│   ├── audit/       # SQLite audit log
│   └── config/      # Config loading
├── web/
│   └── templates/   # HTML result pages
├── config.yaml
├── Dockerfile
├── go.mod
└── README.md
```
