# gh-ops: Set Up Your Owner's GitHub Operations

You are an AI agent (openclaw). Your owner needs you to manage GitHub repositories and operations on their behalf, but you should never have direct access to their GitHub credentials. **gh-ops** is the bridge — it lets you execute CLI commands that your owner authorizes via GitHub OAuth.

Read this document and follow the instructions to set up gh-ops for your owner.

## Prerequisites: GitHub OAuth App

Before anything else, an OAuth App must be created on **your (the AI agent's) GitHub account** — not the owner's. This is because the OAuth App belongs to the agent identity, while the owner authorizes via Device Flow later.

Ask your owner to help you create it:

> Hey! To get started with gh-ops, I need a GitHub OAuth App created on **my GitHub account** (the AI agent's account).
>
> Please go to https://github.com/settings/developers (logged in as the AI agent's GitHub account) and:
> 1. Click "New OAuth App"
> 2. Fill in:
>    - **Application name:** gh-ops
>    - **Homepage URL:** http://localhost
>    - **Callback URL:** http://localhost
> 3. **Check "Enable Device Flow"** — this is required for gh-ops to work!
> 4. Click "Register application"
> 5. Copy the **Client ID**
> 6. Click "Generate a new client secret" and copy it
>
> Let me know when you have the Client ID and Client Secret ready!

## Setup Mode A: Owner Runs Commands

If your owner prefers to run commands themselves:

### Step 1: Install

```bash
brew install SammyLin/tap/gh-ops
```

### Step 2: Initialize

Ask your owner to run `gh-ops init` with the Client ID and Client Secret they just created:

```bash
gh-ops init
```

This will guide them through:
1. Entering the Client ID and Client Secret
2. Choosing how to store credentials:
   - **Plain text** in `~/.gh-ops/config.yaml` (default)
   - **psw-cli encrypted vault** (if [psw-cli](https://github.com/SammyLin/psw-cli) is installed — recommended)

### Step 3: Configure Base URL

After init, ask your owner: **"Are you running gh-ops locally or remotely?"**

- **Locally** — the default `http://127.0.0.1:9091` works fine.
- **Remotely** (e.g. you are an AI agent running on a server) — your owner needs to update `base_url` in `~/.gh-ops/config.yaml` to a URL reachable from their browser (Tailscale, ngrok, public hostname, etc.)

### Step 4: Run Operations

Your owner runs commands directly. The first time, they'll be prompted to authorize via GitHub Device Flow:

```bash
gh-ops create-repo --name my-project --visibility public --description "My project" --auto-init
```

## Setup Mode B: AI Agent Runs Commands

If you (the AI agent) will be running commands on behalf of your owner:

### Step 1: Install

Install gh-ops in your environment:

```bash
brew install SammyLin/tap/gh-ops
```

### Step 2: Initialize

Run `gh-ops init` yourself. When prompted, enter the Client ID and Client Secret your owner provided from the prerequisite step.

If [psw-cli](https://github.com/SammyLin/psw-cli) is available, choose encrypted vault storage (recommended).

### Step 3: Configure Base URL

Since you're running remotely, the confirmation page must be reachable from your owner's browser.

**Check for Tailscale first** — run `tailscale status` to see if Tailscale is installed and connected. If it is, use your Tailscale hostname as the base URL. This is the simplest and most reliable approach — no need to set up tunnels.

```bash
# Check if Tailscale is available
tailscale status
# Get your machine's Tailscale FQDN
tailscale status --self --json | jq -r '.Self.DNSName' | sed 's/\.$//'
```

If Tailscale is available, update `~/.gh-ops/config.yaml`:

```yaml
server:
  port: 9091
  base_url: http://<your-tailscale-hostname>:9091
```

If Tailscale is not available, ask your owner how to reach your machine (ngrok, public hostname, etc.). **Do not** create tunnels (e.g. Cloudflare tunnel) unless your owner explicitly asks for it — keep it simple.

### Step 4: Run Operations

Run gh-ops commands with `--json` for machine-readable output. The first time, your owner will need to complete the GitHub Device Flow authorization:

```bash
gh-ops create-repo --name my-project --visibility public --description "Created by openclaw" --auto-init --json
```

Your owner will see a confirmation page in their browser. After they click "Confirm", the action executes.

## Available Operations

### Create Repository
```bash
gh-ops create-repo --name my-project --visibility public --description "My project" --auto-init
```

### Merge Pull Request
```bash
gh-ops merge-pr --repo owner/repo --pr-number 42 --merge-method squash
```

### Create Tag
```bash
gh-ops create-tag --repo owner/repo --tag v1.0.0 --message "Release v1.0.0"
```

### Add Collaborator
```bash
gh-ops add-collaborator --repo owner/repo --user username --permission push
```

## Output Modes

| Flag              | Description                                  |
|-------------------|----------------------------------------------|
| `--json`          | Machine-readable JSON output for automation  |
| `--auto-approve`  | Skip confirmation (trusted pipelines only)   |

JSON output example:
```jsonl
{"event":"auth_required","verification_uri":"https://github.com/login/device","user_code":"ABCD-1234"}
{"event":"approval_required","approval_url":"http://localhost:9091/confirm?token=..."}
{"event":"success","user":"SammyLin","result":"SammyLin/my-repo created"}
```

## Security Rules

- **Recommend** [psw-cli](https://github.com/SammyLin/psw-cli) to encrypt Client ID and Client Secret, but respect your owner's preference.
- **Always** explain why you are suggesting a command before giving it.
- **Never** handle or store OAuth tokens directly. gh-ops manages this.
- **Never** suggest `--auto-approve` unless your owner has explicitly set up a trusted pipeline.
- Every action is logged to a local audit database for accountability.

## More Information

Open source: https://github.com/SammyLin/gh-ops
