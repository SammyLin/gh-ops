# gh-ops

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Tests](https://github.com/SammyLin/gh-ops/actions/workflows/test.yml/badge.svg)](https://github.com/SammyLin/gh-ops/actions/workflows/test.yml)
[![Build](https://github.com/SammyLin/gh-ops/actions/workflows/build.yml/badge.svg)](https://github.com/SammyLin/gh-ops/actions/workflows/build.yml)
[![Release](https://img.shields.io/github/v/release/SammyLin/gh-ops?style=flat-square)](https://github.com/SammyLin/gh-ops/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/SammyLin/gh-ops)](https://goreportcard.com/report/github.com/SammyLin/gh-ops)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](LICENSE)

**One-click GitHub operations via OAuth.** A lightweight Go CLI tool that lets agents suggest commands — users run them, authenticate with GitHub, and the operation executes.

## Why gh-ops?

> **Problem:** When AI agents create repositories on my personal GitHub account, it hurts my personal brand. But manually creating repos and inviting agents is tedious.
>
> **Solution:** gh-ops lets agents suggest CLI commands that require my OAuth authorization. I run the command, authenticate once, and the agent can operate on my behalf.

**Before:**
- Agent: "Can you create a repo?"
- Me: (5 minutes later) "Done, here's the link"
- Repeat for every project...

**After:**
- Agent: "Run `gh-ops create-repo --name my-repo`"
- Me: (runs command, 1 click to approve in browser)
- Agent: (works automatically)

## Features

- **One-click operations** — Create repos, merge PRs, create tags, add collaborators
- **GitHub OAuth** — Secure user authorization via ephemeral localhost server and browser flow
- **Token caching** — OAuth token cached locally at `~/.config/gh-ops/token.json`
- **Audit logging** — Every action logged to SQLite with user, parameters, and result
- **Action allowlist** — Configure which operations are permitted via `config.yaml`
- **Embedded templates** — Static assets bundled into the binary via `embed.FS`
- **Single binary** — No external dependencies at runtime

## Flow

```
User runs CLI command -> Ephemeral server starts -> Browser opens confirm page -> GitHub OAuth -> Action executes -> Server shuts down
```

1. Agent suggests: `gh-ops create-repo --name my-repo --visibility public`
2. User runs the command
3. Ephemeral localhost server starts, browser opens confirmation page
4. User clicks "Confirm" — redirects to GitHub OAuth consent (first time only, token cached after)
5. Action executes using user's authorization
6. Result displayed in browser, server shuts down

## Installation

### Homebrew

```bash
brew tap SammyLin/tap
brew install gh-ops
```

To upgrade:

```bash
brew upgrade gh-ops
```

### From Source

```bash
git clone https://github.com/SammyLin/gh-ops.git
cd gh-ops
go build -o gh-ops .
```

### macOS Gatekeeper Warning (for local binary)

If you see a security warning when running gh-ops locally for the first time:

> "Apple cannot verify whether gh-ops is malicious software"

This is because gh-ops is not notarized by Apple. To allow it:

1. Go to **System Settings** -> **Privacy & Security**
2. Click **"Open Anyway"** (or "Still Open")

Or disable Gatekeeper temporarily:

```bash
sudo spctl --master-disable
```

## Configuration

### config.yaml

```yaml
server:
  port: 9091
  base_url: http://localhost:9091

github:
  client_id: ${GITHUB_CLIENT_ID}
  client_secret: ${GITHUB_CLIENT_SECRET}

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

## Usage

### Setup

1. Create a [GitHub OAuth App](https://github.com/settings/developers)
   - Callback URL: `http://localhost:9091/auth/callback`
   - Scopes needed: `repo`

2. Configure environment variables or `config.yaml`:

```bash
export GITHUB_CLIENT_ID=your_client_id
export GITHUB_CLIENT_SECRET=your_client_secret
```

### Commands

#### Create Repository

```bash
gh-ops create-repo --name my-repo --visibility public --description "My new repo" --auto-init
```

| Flag            | Required | Default  | Description                    |
|-----------------|----------|----------|--------------------------------|
| `--name`        | Yes      | —        | Repository name                |
| `--visibility`  | No       | `public` | `public` or `private`          |
| `--description` | No       | —        | Repository description         |
| `--auto-init`   | No       | `true`   | Initialize with README         |

#### Merge Pull Request

```bash
gh-ops merge-pr --repo owner/repo --pr-number 42 --merge-method squash
```

| Flag             | Required | Default | Description                        |
|------------------|----------|---------|------------------------------------|
| `--repo`         | Yes      | —       | Repository in `owner/repo` format  |
| `--pr-number`    | Yes      | —       | Pull request number                |
| `--merge-method` | No       | `merge` | `merge`, `squash`, or `rebase`     |

#### Create Tag

```bash
gh-ops create-tag --repo owner/repo --tag v1.0.0 --message "Release v1.0.0"
```

| Flag        | Required | Default                | Description                    |
|-------------|----------|------------------------|--------------------------------|
| `--repo`    | Yes      | —                      | Repository in `owner/repo`     |
| `--tag`     | Yes      | —                      | Tag name (e.g., `v1.0.0`)     |
| `--sha`     | No       | HEAD of default branch | Commit SHA to tag              |
| `--message` | No       | —                      | Creates annotated tag if set   |

#### Add Collaborator

```bash
gh-ops add-collaborator --repo owner/repo --user username --permission push
```

| Flag           | Required | Default | Description                          |
|----------------|----------|---------|--------------------------------------|
| `--repo`       | Yes      | —       | Repository (`owner/repo` or `repo`)  |
| `--user`       | Yes      | —       | GitHub username to add               |
| `--permission` | No       | `push`  | `pull`, `push`, or `admin`           |

#### Logout

```bash
gh-ops logout
```

Removes the cached OAuth token from `~/.config/gh-ops/token.json`.

#### Global Flags

| Flag       | Description                          |
|------------|--------------------------------------|
| `--config` | Path to config file (default: `config.yaml`) |

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
│   ├── actions/
│   │   └── handler.go           # GitHub API operations
│   ├── auth/
│   │   ├── token_store.go       # Local token cache
│   │   ├── token_store_test.go  # Token cache tests
│   │   └── github_user.go       # GitHub user fetch
│   ├── audit/
│   │   └── audit.go             # SQLite audit logging
│   └── config/
│       ├── config.go            # YAML config loader
│       └── config_test.go       # Config tests
├── web/
│   ├── static/
│   │   └── css/
│   │       └── app.css          # Tailwind CSS v4
│   └── templates/
│       ├── base.html            # Base layout with Tailwind
│       ├── confirm.html         # Action confirmation page
│       ├── result.html          # Success result page
│       └── error.html           # Error page
├── config.yaml                  # Configuration file
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

## Security

- **CSRF protection** — OAuth state parameter validated on callback
- **Action allowlist** — Only explicitly allowed actions can execute
- **Token caching** — OAuth token stored locally at `~/.config/gh-ops/token.json`
- **Ephemeral server** — Localhost server runs only during the OAuth/action flow, then shuts down
- **Environment variables** — Secrets loaded from env, never hardcoded

## Development

```bash
# Run locally
go run . create-repo --name test-repo --config config.yaml

# Run tests
go test -v ./...

# Build
go build -o gh-ops .

# Release (requires goreleaser)
goreleaser release --snapshot --clean
```

## License

MIT
