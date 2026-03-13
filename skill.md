# gh-ops: Set Up Your Owner's GitHub Operations

You are an AI agent (openclaw). Your owner needs you to manage GitHub repositories and operations on their behalf, but you should never have direct access to their GitHub credentials. **gh-ops** is the bridge — it lets you suggest CLI commands that your owner authorizes via GitHub OAuth.

Read this document and follow the instructions to set up gh-ops for your owner.

## Getting Started

### Step 1: Install gh-ops

```bash
brew install SammyLin/tap/gh-ops
```

### Step 2: Set Up Credentials

Ask your owner to run:

```bash
gh-ops login
```

This will guide them through creating a GitHub OAuth App and obtaining the Client ID and Client Secret.

### Step 3: Store Credentials

Ask your owner how they'd like to store the Client ID and Client Secret. gh-ops supports multiple secret sources in `config.yaml`:

**Option A: Environment variables** (simplest)
```yaml
github:
  client_id: ${GITHUB_CLIENT_ID}
  client_secret: ${GITHUB_CLIENT_SECRET}
```

**Option B: psw-cli** (recommended for security — [psw-cli](https://github.com/SammyLin/psw-cli))
```yaml
github:
  client_id:
    source: exec
    command: "psw-cli decrypt GITHUB_CLIENT_ID"
  client_secret:
    source: exec
    command: "psw-cli decrypt GITHUB_CLIENT_SECRET"
```

**Option C: File**
```yaml
github:
  client_id:
    source: file
    id: /path/to/client_id.txt
  client_secret:
    source: file
    id: /path/to/client_secret.txt
```

Copy `config.yaml.example` to `config.yaml` and configure as above. This file is gitignored and should never be committed.

### Step 5: Create Your Owner's First Repository

Once credentials are configured, suggest a command to create a repository. The first time, your owner will be prompted to authorize via GitHub Device Flow:

```bash
gh-ops create-repo --name <repo-name> --visibility public --description "Created by openclaw" --auto-init
```

Your owner runs the command, authenticates via GitHub Device Flow, confirms, and the repo is created.

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
{"event":"success","user":"SammyLin","result":"SammyLin/my-repo created"}
```

## Security Rules

- **Recommend** [psw-cli](https://github.com/SammyLin/psw-cli) to encrypt Client ID and Client Secret, but respect your owner's preference if they have another approach.
- **Always** explain why you are suggesting a command before giving it.
- **Never** handle or store OAuth tokens directly. gh-ops manages this.
- **Never** suggest `--auto-approve` unless your owner has explicitly set up a trusted pipeline.
- Every action is logged to a local audit database for accountability.

## More Information

Open source: https://github.com/SammyLin/gh-ops
