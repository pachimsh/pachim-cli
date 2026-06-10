# Pachim CLI

Deploy projects from your terminal to sites on [Pachim](https://pachim.sh).

The CLI packages your local project, uploads it to Pachim, and monitors the deployment. It supports **multiple sites per project**, **automatic target selection by git branch**, and **git-merge uploads** (incremental deploys instead of full replacements).

## Installation

### Quick install (recommended)

**Linux / macOS**

```bash
curl -fsSL https://mirrors.pachim.app/cli/install.sh | sh
```

**Windows (PowerShell)**

```powershell
irm https://mirrors.pachim.app/cli/install.ps1 | iex
```

Install scripts try **mirrors.pachim.app** first, then fall back to **GitHub** if the mirror is unavailable.

**Windows (MSI)** — download `pachim_windows_amd64.msi` from [GitHub Releases](https://github.com/pachimsh/pachim-cli/releases). Installs to `C:\Program Files\Pachim` and adds it to `PATH`.

### Package managers

| Platform | Command                                                                                      |
| -------- | -------------------------------------------------------------------------------------------- |
| Homebrew | `brew install pachimsh/homebrew-tap/pachim`                                                  |
| Scoop    | `scoop bucket add pachimsh https://github.com/pachimsh/scoop-bucket && scoop install pachim` |
| WinGet   | `winget install Pachim.PachimCLI`                                                            |

### Build from source

```bash
git clone https://github.com/pachimsh/pachim-cli.git
cd pachim-cli
go build -o pachim .
```

---

## Quick start

```bash
# 1. Authenticate
pachim login

# 2. Link this project to your Pachim site(s)
pachim init

# 3. Sync branch mappings (first time, or after upgrading CLI)
pachim link sync

# 4. Deploy
pachim push
```

---

## Typical workflow

### Single site

```bash
pachim init          # pick your site
pachim link sync     # set deploy_branch (or pull from Pachim)
pachim push          # package, upload, watch deployment
```

### Staging + production (one repo, multiple branches)

Map each git branch to a site in `.pachim.json`. After that, **`pachim push` picks the target from your current branch** — no need to re-run `init` or switch configs manually.

```bash
git checkout develop
pachim push          # → staging (e.g. dapi.example.com)

git checkout main
pachim push          # → production (e.g. api.example.com)
```

Preview before deploying:

```bash
pachim status
# or
pachim push --dry-run
```

---

## Commands

### Authentication & accounts

| Command           | Description                                     |
| ----------------- | ----------------------------------------------- |
| `pachim login`    | Sign in (token stored in `~/.pachim/profiles/`) |
| `pachim logout`   | Remove stored credentials                       |
| `pachim whoami`   | Show the logged-in user and active workspace      |
| `pachim profiles` | List saved profiles                               |

### Workspaces

Team accounts scope sites and servers by **workspace**. The CLI sends `X-Workspace-Id` on every authenticated request so `pachim sites`, `init`, and `push` see the same resources as the dashboard.

| Command                    | Description                                              |
| -------------------------- | -------------------------------------------------------- |
| `pachim workspace list`    | List workspaces you can access                           |
| `pachim workspace use <id\|slug>` | Pin the active workspace for this profile         |
| `pachim workspace use personal`   | Reset to your personal workspace (default)        |
| `pachim workspace current` | Show selected and API-resolved workspace context         |

```bash
pachim workspace list
pachim workspace use my-team-slug
pachim sites          # sites from that workspace
pachim init           # link sites from that workspace
```

Priority for workspace selection:

1. `--workspace <id|slug>` global flag
2. `PACHIM_WORKSPACE_ID` or `PACHIM_WORKSPACE` environment variable
3. `workspace_id` in `.pachim.json` (saved by `pachim init`; accepts id or slug)
4. Profile preference from `pachim workspace use`
5. Personal workspace (API default when nothing is set)

Use multiple accounts with `--profile` or `PACHIM_PROFILE`:

```bash
pachim --profile work login
pachim --profile work push
```

### Project setup

| Command              | Description                                                                                                             |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| `pachim init`        | Link the current directory to one or more Pachim sites; creates `.pachim.json`                                          |
| `pachim link sync`   | Sync sites with Pachim (remove deleted sites, pull `deploy_branch` from server, interactive setup for missing mappings) |
| `pachim link show`   | Show configured sites, branch mappings, and which site matches the current git branch                                   |
| `pachim use [alias]` | Set the default site alias in `.pachim.json`                                                                            |

### Deploy

| Command              | Description                                      |
| -------------------- | ------------------------------------------------ |
| `pachim push`        | Package, upload, and deploy to the resolved site |
| `pachim status`      | Show the deploy plan without uploading           |
| `pachim deployments` | List recent deployments for a site               |

### Site management

| Command            | Description                                        |
| ------------------ | -------------------------------------------------- |
| `pachim sites`     | List all sites on your Pachim account              |
| `pachim git-merge` | View or toggle git-merge mode for the default site |

### Maintenance

| Command              | Description                          |
| -------------------- | ------------------------------------ |
| `pachim self-update` | Update the CLI to the latest release |

### Global flags

| Flag               | Description                                               |
| ------------------ | --------------------------------------------------------- |
| `--profile <name>` | Use a specific credentials profile                        |
| `--workspace <id\|slug>` | Override active workspace for this command                |
| `--verbose`        | Show sync details, branch matching, and extra deploy info |
| `--quiet`          | Minimal output (errors and final results only)            |

---

## `pachim push`

Packages files from the working directory (respecting `.gitignore` via `git ls-files`), uploads a zip, and streams deployment logs until finished.

### Target selection

Priority order:

1. `--site <alias>` — explicit override
2. **Git branch mapping** — if current branch matches exactly one site's `deploy_branch`
3. Single site in config — used automatically
4. Interactive site picker — when ambiguous

### Flags

| Flag              | Description                                                                          |
| ----------------- | ------------------------------------------------------------------------------------ |
| `--site <alias>`  | Deploy to a specific site                                                            |
| `--branch <name>` | Branch metadata sent to Pachim (defaults to mapped / current branch)                 |
| `--save-branch`   | Save the deploy branch as the site default on Pachim                                 |
| `-y`, `--yes`     | Skip deploy confirmation (for CI); still respects production safety unless `--force` |
| `--dry-run`       | Print the deploy plan and exit                                                       |
| `--force`         | Allow deploy with production branch + uncommitted local changes                      |

### Deploy plan

When confirmation is needed, the CLI shows a single **deploy plan** instead of multiple separate prompts:

- Target site and domain
- Branch and commit
- Git state (clean / uncommitted changes)
- Git-merge vs full-replace mode
- Active deployment on server (watch, queue, or cancel)

On a **happy path** (branch match, clean tree, git-merge on, no active deploy), push runs with one line:

```text
→ Deploying api.example.com (main @ a1b2c3d)...
```

### Production safety

Deploying a production branch (`main`, `master`, `production`, `prod`) with **uncommitted changes** triggers an extra confirmation. Use `--force` only when you intend to upload local changes to production.

### CI example

```bash
pachim --profile production \
  --workspace team-alpha \
  push \
  --site production \
  --branch "$GITHUB_REF_NAME" \
  --yes
```

Ensure `.pachim.json` and branch mappings exist in the repo (or run `pachim link sync` in a setup step).

---

## Git merge

When **git merge** is enabled on a site, uploads are merged with existing code on the server (similar to `git push`). When disabled, each upload **replaces** all code.

- During push, you can enable git merge from the deploy plan
- After the first successful deploy, the CLI may offer to enable it
- Manage manually: `pachim git-merge`, `pachim git-merge --enable`, `pachim git-merge --disable`

---

## Configuration

### Credentials

Stored at `~/.pachim/profiles/<name>.json` (mode `0600`). **Never commit this file.**

### Project config — `.pachim.json`

Created by `pachim init`. Automatically added to `.gitignore` when inside a git repo.

```json
{
  "workspace_id": "01abc...",
  "default": "production",
  "sites": {
    "staging": {
      "site_id": "01abc...",
      "domain": "dapi.example.com",
      "deploy_branch": "develop",
      "label": "Staging API"
    },
    "production": {
      "site_id": "01xyz...",
      "domain": "api.example.com",
      "deploy_branch": "main",
      "label": "Production API"
    }
  }
}
```

| Field           | Description                                                       |
| --------------- | ----------------------------------------------------------------- |
| `workspace_id`  | Optional workspace scope for this project (id or slug; set by `pachim init`) |
| `default`       | Fallback site alias (used when branch mapping does not match)     |
| `site_id`       | Pachim site ID                                                    |
| `domain`        | Site domain                                                       |
| `deploy_branch` | Git branch mapped to this site; drives automatic target selection |
| `label`         | Optional display name in CLI prompts                              |

### Keeping config in sync

On each `pachim push`:

- Sites deleted from Pachim are **removed** from `.pachim.json`
- Missing `deploy_branch` values are **pulled from Pachim** when available

If mappings are still incomplete:

```bash
pachim link sync
```

### Upgrading from an older CLI

If your `.pachim.json` has sites but no `deploy_branch`:

```bash
pachim link sync
```

The CLI pulls branches from Pachim first, then asks only for sites that still need a mapping.

---

## Environment variables

| Variable              | Description                                              |
| --------------------- | -------------------------------------------------------- |
| `PACHIM_PROFILE`      | Default profile name                                     |
| `PACHIM_WORKSPACE_ID` | Active workspace id or slug for CLI requests               |
| `PACHIM_WORKSPACE`    | Alias for workspace slug (used when `PACHIM_WORKSPACE_ID` is unset) |
| `PACHIM_API_URL`      | Override API base URL (default: `https://api.pachim.sh`) |

---

## Development

### Custom API URL

```bash
export PACHIM_API_URL=http://localhost:8000
pachim login

# or per command
pachim --api-url http://localhost:8000 login
```

### Build

```bash
go build -o pachim .

# cross-compile
GOOS=linux GOARCH=amd64 go build -o pachim-linux .
GOOS=darwin GOARCH=arm64 go build -o pachim-mac .
GOOS=windows GOARCH=amd64 go build -o pachim.exe .
```

### Release

Tags trigger GoReleaser via GitHub Actions:

```bash
git tag v0.4.0
git push origin v0.4.0
```

For mirror deployment to `mirrors.pachim.app`, see [docs/DEPLOY.md](docs/DEPLOY.md).

---

## License

See [LICENSE](LICENSE).
