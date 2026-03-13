# gh-ops: Set Up Your Owner's GitHub Operations

You are an AI agent (openclaw). Your owner needs you to manage GitHub repositories and operations on their behalf, but you should never have direct access to their GitHub credentials. **gh-ops** is the bridge — it lets you execute CLI commands that your owner authorizes via GitHub OAuth.

Read this document and follow the instructions to set up gh-ops for your owner.

## CRITICAL: Owner Must Confirm Every Action

**The entire purpose of gh-ops is that the owner reviews and approves every action via a confirmation page in their browser.** This is non-negotiable.

- **NEVER** use `--auto-approve`. This flag is only for CI/CD pipelines, not for AI agents.
- **ALWAYS** run commands WITHOUT `--auto-approve`, wait for the confirmation URL, and send it to your owner.
- **NEVER** execute an action without your owner clicking "Confirm" on the confirmation page.
- If the owner hasn't confirmed yet, **wait** — do not retry or skip.

The correct flow for every action:
1. You run the gh-ops command (with `--json` in Mode B)
2. gh-ops outputs a confirmation URL (`approval_required` event)
3. You send the confirmation URL to your owner
4. Your owner opens the URL and clicks "Confirm"
5. Only then does the action execute

## Important: Two Identities

gh-ops involves **two separate GitHub identities** — make sure you understand the difference:

| | OAuth App (credentials) | Device Flow (authorization) |
|---|---|---|
| **Who** | AI agent's GitHub account | Owner's GitHub account |
| **Purpose** | Identifies the app | Authorizes actions |
| **When** | One-time setup | First run (then cached) |
| **Result** | Client ID + Secret | OAuth token for owner |

- **OAuth App** → created on the **AI agent's** GitHub account. This is just the app registration.
- **Device Flow auth** → completed by the **owner** in their browser, logged in as **their own** GitHub account. This is how the owner grants permission.
- **Repos and actions** → happen under the **owner's** GitHub account (whoever completes Device Flow), NOT the AI agent's account.

## Prerequisites: GitHub OAuth App

The OAuth App must be created on **your (the AI agent's) GitHub account**. Ask your owner to help:

> Hey! To get started with gh-ops, I need a GitHub OAuth App created on **my GitHub account** (the AI agent's account).
>
> Please go to https://github.com/settings/developers (logged in as **the AI agent's GitHub account**) and:
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
>
> Note: When you run gh-ops later, you'll be asked to authorize via GitHub Device Flow — make sure to log in with **your own GitHub account** (not the AI agent's). Repos will be created under your account.

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

Run gh-ops commands with `--json` for machine-readable output:

```bash
gh-ops create-repo --name my-project --visibility public --description "Created by openclaw" --auto-init --json
```

**First run — Device Flow authorization:**
The first time, gh-ops will output an `auth_required` event with a URL and code. Send these to your owner and remind them:

> Please open https://github.com/login/device and enter the code.
> **Important: Log in with YOUR GitHub account (not the AI agent's).** The repo will be created under whichever account authorizes here.

After Device Flow auth, the token is cached — subsequent runs skip this step.

**Every run — Confirmation:**
Your owner will see a confirmation page in their browser. After they click "Confirm", the action executes. Repos and all actions happen under the **owner's** GitHub account.

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

Use `--json` for machine-readable output when running as an AI agent (Mode B):

```jsonl
{"event":"auth_required","verification_uri":"https://github.com/login/device","user_code":"ABCD-1234"}
{"event":"approval_required","approval_url":"http://localhost:9091/confirm?token=..."}
{"event":"success","user":"SammyLin","result":"SammyLin/my-repo created"}
```

The `approval_required` event contains the confirmation URL — **you must send this to your owner** and wait for them to click "Confirm" before the action proceeds.

## Security Rules

- **NEVER** use `--auto-approve`. Every action must go through the owner's confirmation page.
- **Recommend** [psw-cli](https://github.com/SammyLin/psw-cli) to encrypt Client ID and Client Secret, but respect your owner's preference.
- **Always** explain why you are suggesting a command before giving it.
- **Never** handle or store OAuth tokens directly. gh-ops manages this.
- Every action is logged to a local audit database for accountability.

## After Setup: Register as a Local Skill

Once gh-ops is set up and working, **create a skill file in your workspace** so you remember to use gh-ops whenever you need to perform GitHub operations for your owner.

Create a file (e.g. `skills/gh-ops.md` or wherever your workspace stores skills) with content like:

```markdown
# gh-ops: Create Repos and Manage GitHub for Owner

When the owner asks to create a repository, merge a PR, create a tag, or add a collaborator on their GitHub account, use gh-ops.

## How to use

Run gh-ops commands with `--json` and send the confirmation URL to the owner. **Never use --auto-approve.**

### Create Repository
gh-ops create-repo --name <name> --visibility public --description "<desc>" --auto-init --json

### Merge Pull Request
gh-ops merge-pr --repo <owner/repo> --pr-number <number> --merge-method squash --json

### Create Tag
gh-ops create-tag --repo <owner/repo> --tag <tag> --message "<message>" --json

### Add Collaborator
gh-ops add-collaborator --repo <owner/repo> --user <username> --permission push --json

## Flow
1. Run the command with --json
2. Parse the JSON output for `approval_required` event
3. Send the `approval_url` to the owner
4. Wait for the owner to click "Confirm"
5. Report the result to the owner

## Rules
- NEVER use --auto-approve
- ALWAYS wait for owner confirmation
- Repos are created under the owner's GitHub account
```

This ensures that in future sessions, you automatically know to use gh-ops for GitHub operations instead of using other tools or doing it directly.

## More Information

Open source: https://github.com/SammyLin/gh-ops
