# gh-ops

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Tests](https://github.com/SammyLin/gh-ops/actions/workflows/test.yml/badge.svg)](https://github.com/SammyLin/gh-ops/actions/workflows/test.yml)
[![Build](https://github.com/SammyLin/gh-ops/actions/workflows/build.yml/badge.svg)](https://github.com/SammyLin/gh-ops/actions/workflows/build.yml)
[![Release](https://img.shields.io/github/v/release/SammyLin/gh-ops?style=flat-square)](https://github.com/SammyLin/gh-ops/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/SammyLin/gh-ops)](https://goreportcard.com/report/github.com/SammyLin/gh-ops)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](LICENSE)

**One-click GitHub operations via OAuth.** A lightweight Go CLI tool that lets agents suggest commands — users run them, authenticate with GitHub Device Flow, and the operation executes after web confirmation.

## Why gh-ops?

> **Problem:** When AI agents create repositories on my personal GitHub account, it hurts my personal brand. But manually creating repos and inviting agents is tedious.
>
> **Solution:** gh-ops lets agents suggest CLI commands that require my OAuth authorization. I run the command, authenticate once, and the agent can operate on my behalf.

## Features

- **Web confirmation** — Every action requires user approval on a confirmation page before executing
- **GitHub Device Flow** — Secure OAuth via GitHub's Device Flow (no browser redirect needed)
- **Token caching** — OAuth token cached locally at `~/.config/gh-ops/token.json`
- **JSON output** — Machine-readable output for bot/automation integrations (e.g., Telegram)
- **Auto-approve mode** — Skip confirmation with `--auto-approve` for trusted pipelines
- **Audit logging** — Every action logged to SQLite with user, parameters, and result
- **Action allowlist** — Configure which operations are permitted via `config.yaml`
- **Single binary** — No external dependencies at runtime

## Flow

```
User runs CLI command
  -> GitHub Device Flow auth (first time, then cached)
  -> Confirmation page served on localhost
  -> User clicks "Confirm"
  -> Action executes
  -> Server shuts down
```

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

## Configuration

### config.yaml

```yaml
server:
  port: 9091
  base_url: http://127.0.0.1:9091

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

| Variable               | Description                     |
|------------------------|---------------------------------|
| `GITHUB_CLIENT_ID`     | GitHub OAuth App client ID      |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth App client secret  |

### Setup

1. Create a [GitHub OAuth App](https://github.com/settings/developers)
   - No callback URL needed (uses Device Flow)
   - Scopes needed: `repo`

2. Set the environment variable:

```bash
export GITHUB_CLIENT_ID=your_client_id
export GITHUB_CLIENT_SECRET=your_client_secret
```

## Usage

### Global Flags

| Flag              | Description                                    |
|-------------------|------------------------------------------------|
| `--config`        | Path to config file (default: `config.yaml`)   |
| `--json`          | Output in JSON format (for bot integrations)   |
| `--auto-approve`  | Skip web confirmation, execute immediately     |

### Usage Modes

| Command | Behavior |
|---------|----------|
| `gh-ops create-repo --name x` | Auth -> Confirmation page -> Execute |
| `gh-ops create-repo --name x --auto-approve` | Auth -> Execute immediately |
| `gh-ops create-repo --name x --json` | JSON output + Confirmation page (for Telegram/bots) |
| `gh-ops create-repo --name x --json --auto-approve` | JSON output + Execute immediately |

### JSON Output

When using `--json`, gh-ops outputs newline-delimited JSON events:

```jsonl
{"event":"auth_required","verification_uri":"https://github.com/login/device","user_code":"ABCD-1234"}
{"event":"approval_required","approval_url":"http://localhost:9091/confirm?token=..."}
{"event":"success","user":"SammyLin","result":"SammyLin/my-repo created"}
```

| Event               | Description                                       |
|---------------------|---------------------------------------------------|
| `auth_required`     | User needs to visit URL and enter code             |
| `approval_required` | User needs to open URL and click Confirm           |
| `success`           | Action completed successfully                      |
| `error`             | Action failed                                      |

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

- **Action allowlist** — Only explicitly allowed actions can execute
- **Web confirmation** — Every action requires explicit user approval (unless `--auto-approve`)
- **Token caching** — OAuth token stored locally at `~/.config/gh-ops/token.json` with `0600` permissions
- **Ephemeral server** — Confirmation server runs only during the approval flow, then shuts down
- **Environment variables** — Secrets loaded from env, never hardcoded

## For AI Agents

Read https://raw.githubusercontent.com/SammyLin/gh-ops/refs/heads/main/skill.md and follow the instructions to set up gh-ops for your owner.

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
