# gh-ops

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Tests](https://github.com/SammyLin/gh-ops/actions/workflows/test.yml/badge.svg)](https://github.com/SammyLin/gh-ops/actions/workflows/test.yml)
[![Build](https://github.com/SammyLin/gh-ops/actions/workflows/build.yml/badge.svg)](https://github.com/SammyLin/gh-ops/actions/workflows/build.yml)
[![Release](https://img.shields.io/github/v/release/SammyLin/gh-ops?style=flat-square)](https://github.com/SammyLin/gh-ops/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/SammyLin/gh-ops)](https://goreportcard.com/report/github.com/SammyLin/gh-ops)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](LICENSE)

**One-click GitHub operations via OAuth.** A lightweight Go web service that lets agents generate action URLs — users click, authenticate with GitHub, and the operation executes.

## Why gh-ops?

> **Problem:** When AI agents create repositories on my personal GitHub account, it hurts my personal brand. But manually creating repos and inviting agents is tedious.
>
> **Solution:** gh-ops lets agents generate action URLs that require my OAuth authorization. I click, authenticate once, and the agent can operate on my behalf.

**Before:**
- Agent: "Can you create a repo?"
- Me: (5 minutes later) "Done, here's the link"
- Repeat for every project...

**After:**
- Agent: "Here's the link, please authorize"
- Me: (2 clicks) "Approved"
- Agent: (works automatically)

## Features

- **One-click operations** — Create repos, merge PRs, create tags, add collaborators
- **GitHub OAuth** — Secure user authorization with encrypted cookie sessions
- **Audit logging** — Every action logged to SQLite with user, parameters, and result
- **Rate limiting** — Per-IP request throttling to prevent abuse
- **Action allowlist** — Configure which operations are permitted via `config.yaml`
- **Embedded templates** — Static assets bundled into the binary via `embed.FS`
- **Single binary** — No external dependencies at runtime

## Flow

```
Agent generates URL → User clicks → GitHub OAuth → Action executes → Result page
```

1. Agent generates link: `https://ghops.example.com/action/create-repo?name=my-repo`
2. User clicks the link
3. Redirects to GitHub OAuth consent (first time only)
4. Action executes using user's authorization
5. Shows result page with outcome

## Installation

### Homebrew

```bash
brew install SammyLin/tap/gh-ops
```

### From Source

```bash
git clone https://github.com/SammyLin/gh-ops.git
cd gh-ops
go build -o gh-ops .
```

### Docker

```bash
docker build -t gh-ops .
docker run -p 8080:8080 \
  -e GITHUB_CLIENT_ID=xxx \
  -e GITHUB_CLIENT_SECRET=xxx \
  -e SESSION_SECRET=xxx \
  gh-ops
```

### Homebrew

```bash
brew tap SammyLin/tap
brew install gh-ops
```

To upgrade:

```bash
brew upgrade gh-ops
```

### ⚠️ macOS Gatekeeper Warning (for local binary)

If you see a security warning when running gh-ops locally for the first time:

> "Apple cannot verify whether gh-ops is malicious software"

This is because gh-ops is not notarized by Apple. To allow it:

1. Go to **System Settings** → **Privacy & Security**
2. Click **"Open Anyway"** (or "Still Open")

Or disable Gatekeeper temporarily:

```bash
sudo spctl --master-disable
```

## Configuration

### Environment Variables

Copy `.env.example` to `.env` and fill in your values:

```bash
cp .env.example .env
# Edit .env with your values
```

Required variables:
- `GITHUB_CLIENT_ID` - Your GitHub OAuth App Client ID
- `GITHUB_CLIENT_SECRET` - Your GitHub OAuth App Client Secret
- `SESSION_SECRET` - Generate with: `openssl rand -hex 32`

Optional variables:
- `PORT` - Server port (default: 9091)
- `BASE_URL` - Public URL for OAuth callbacks

### config.yaml

Alternatively, use `config.yaml`:

```yaml
server:
  port: 9091
  base_url: http://localhost:9091

github:
  client_id: ${GITHUB_CLIENT_ID}
  client_secret: ${GITHUB_CLIENT_SECRET}

session:
  secret: ${SESSION_SECRET}

allowed_actions:
  - create-repo
  - merge-pr
  - create-tag
  - add-collaborator
```

## Usage

### Setup

1. Create a [GitHub OAuth App](https://github.com/settings/developers)
   - Callback URL: `https://your-domain/auth/callback`
   - Scopes needed: `repo`

2. Configure environment variables or `config.yaml`:

```bash
export GITHUB_CLIENT_ID=your_client_id
export GITHUB_CLIENT_SECRET=your_client_secret
export SESSION_SECRET=your_session_secret
```

3. Start the server:

```bash
./gh-ops --config config.yaml
```

### Actions

Generate a URL and share it. When the user clicks, they authenticate via GitHub OAuth (first time only) and the action executes.

#### Create Repository

```
GET /action/create-repo?name=my-repo&visibility=public&description=My+new+repo
```

| Parameter     | Required | Default  | Description                    |
|---------------|----------|----------|--------------------------------|
| `name`        | Yes      | —        | Repository name                |
| `visibility`  | No       | `public` | `public` or `private`          |
| `description` | No       | —        | Repository description         |
| `auto_init`   | No       | `true`   | Initialize with README         |

#### Merge Pull Request

```
GET /action/merge-pr?repo=owner/repo&pr_number=42&merge_method=squash
```

| Parameter      | Required | Default | Description                        |
|----------------|----------|---------|------------------------------------|
| `repo`         | Yes      | —       | Repository in `owner/repo` format  |
| `pr_number`    | Yes      | —       | Pull request number                |
| `merge_method` | No       | `merge` | `merge`, `squash`, or `rebase`     |

#### Create Tag

```
GET /action/create-tag?repo=owner/repo&tag=v1.0.0&message=Release+v1.0.0
```

| Parameter | Required | Default                | Description                    |
|-----------|----------|------------------------|--------------------------------|
| `repo`    | Yes      | —                      | Repository in `owner/repo`     |
| `tag`     | Yes      | —                      | Tag name (e.g., `v1.0.0`)     |
| `sha`     | No       | HEAD of default branch | Commit SHA to tag              |
| `message` | No       | —                      | Creates annotated tag if set   |

#### Add Collaborator

```
GET /action/add-collaborator?repo=owner/repo&user=username&permission=push
```

| Parameter    | Required | Default | Description                          |
|--------------|----------|---------|--------------------------------------|
| `repo`       | Yes      | —       | Repository (`owner/repo` or `repo`)  |
| `user`       | Yes      | —       | GitHub username to add               |
| `permission` | No       | `push`  | `pull`, `push`, or `admin`           |

## API Reference

### Public Routes

| Method | Path             | Description               |
|--------|------------------|---------------------------|
| GET    | `/`              | Landing page              |
| GET    | `/health`        | Health check endpoint     |
| GET    | `/auth/login`    | Initiate GitHub OAuth     |
| GET    | `/auth/callback` | OAuth callback handler    |
| GET    | `/auth/logout`   | Clear session and logout  |

### Protected Routes (require OAuth)

| Method | Path               | Description            |
|--------|--------------------|------------------------|
| GET    | `/action/{action}` | Execute GitHub action   |

## Configuration

### config.yaml

```yaml
server:
  port: 8080
  base_url: https://your-domain.com

github:
  client_id: ${GITHUB_CLIENT_ID}
  client_secret: ${GITHUB_CLIENT_SECRET}

session:
  secret: ${SESSION_SECRET}

allowed_actions:
  - create-repo
  - merge-pr
  - create-tag
  - add-collaborator

audit:
  db_path: ./audit.db
```

Environment variables are expanded in the config file using `${VAR}` syntax.

### Environment Variables

| Variable              | Description                      | Default              |
|-----------------------|----------------------------------|----------------------|
| `GITHUB_CLIENT_ID`    | GitHub OAuth App client ID       | —                    |
| `GITHUB_CLIENT_SECRET`| GitHub OAuth App client secret   | —                    |
| `SESSION_SECRET`      | Secret for session encryption    | —                    |

## Directory Structure

```
gh-ops/
├── main.go                      # Entry point, embed templates
├── cmd/
│   └── server.go                # HTTP server setup and routing
├── internal/
│   ├── actions/
│   │   └── handler.go           # GitHub API operations
│   ├── auth/
│   │   └── oauth.go             # GitHub OAuth flow
│   ├── audit/
│   │   └── audit.go             # SQLite audit logging
│   ├── config/
│   │   └── config.go            # YAML config loader
│   └── middleware/
│       └── ratelimit.go         # Per-IP rate limiting
├── web/
│   ├── static/
│   │   └── css/
│   │       └── app.css          # Tailwind CSS v4
│   └── templates/
│       ├── base.html            # Base layout with Tailwind
│       ├── home.html            # Landing page
│       ├── result.html          # Success result page
│       └── error.html           # Error page
├── config.yaml                  # Configuration file
├── Dockerfile                   # Container build
├── .goreleaser.yml              # Release automation
└── README.md
```

## Audit Log

Every action is logged to SQLite:

| Field         | Description                           |
|---------------|---------------------------------------|
| `timestamp`   | When the action occurred              |
| `github_user` | Authenticated GitHub user             |
| `action`      | Action type (e.g., `create-repo`)     |
| `parameters`  | Action parameters (JSON)              |
| `result`      | Success or error message              |
| `ip_address`  | Client IP address                     |

## Security

- **Encrypted sessions** — OAuth tokens stored in AES-256 encrypted cookie sessions
- **CSRF protection** — OAuth state parameter validated on callback
- **Rate limiting** — Per-IP request throttling (60 req/min)
- **Action allowlist** — Only explicitly allowed actions can execute
- **HTTPS** — Deploy behind TLS (e.g., Cloudflare Tunnel, nginx)
- **Environment variables** — Secrets loaded from env, never hardcoded

## Development

```bash
# Run locally
go run . --config config.yaml

# Run tests
go test -v ./...

# Build
go build -o gh-ops .

# Release (requires goreleaser)
goreleaser release --snapshot --clean
```

## Commands

gh-ops is a web service, not a CLI. Use these endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Home page |
| `/health` | GET | Health check |
| `/auth/login` | GET | GitHub OAuth login |
| `/auth/logout` | GET | Logout |
| `/auth/callback` | GET | OAuth callback |
| `/action/create-repo` | GET | Create repository |
| `/action/merge-pr` | GET | Merge pull request |
| `/action/create-tag` | GET | Create git tag |
| `/action/add-collaborator` | GET | Add collaborator |

## License

MIT
