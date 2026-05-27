# Release & Mirror Deployment

## How CI/CD works

1. Push a version tag: `git tag v0.1.0 && git push origin v0.1.0`
2. GitHub Actions runs `.github/workflows/release.yml`
3. GoReleaser builds 6 binaries and publishes a GitHub Release
4. Optionally uploads the same files to `pachim.sh` (Iran-accessible mirror)

## GitHub repository setup

### Required repositories

- `pachim/cli` — main CLI repo
- `pachim/homebrew-tap` — empty repo (GoReleaser pushes formula)
- `pachim/scoop-bucket` — empty repo (GoReleaser pushes manifest)

### Workflow permissions

In `pachim/cli` → **Settings → Actions → General → Workflow permissions**:

- Enable **Read and write permissions**

### Secret for Homebrew / Scoop (recommended)

Create a Personal Access Token with `repo` scope, then add:

| Secret | Value |
|--------|--------|
| `GH_PAT` | your GitHub PAT |

GoReleaser uses `GH_PAT` when set, otherwise `github.token`.

## pachim.sh mirror (Iran / internal network)

### 1. Server directory layout

On your web server (nginx/apache serving `pachim.sh`):

```
/var/www/pachim.sh/cli/          ← adjust path to match your vhost
├── install.sh
├── install.ps1
├── latest.txt                   ← contains e.g. v0.1.0
├── v0.1.0/
│   ├── pachim_linux_amd64.tar.gz
│   ├── pachim_linux_arm64.tar.gz
│   ├── pachim_darwin_amd64.tar.gz
│   ├── pachim_darwin_arm64.tar.gz
│   ├── pachim_windows_amd64.zip
│   ├── pachim_windows_arm64.zip
│   └── checksums.txt
└── v0.1.1/
    └── ...
```

Public URLs:

- `https://pachim.sh/cli/install.sh`
- `https://pachim.sh/cli/latest.txt`
- `https://pachim.sh/cli/v0.1.0/pachim_linux_amd64.tar.gz`

### 2. Nginx example

```nginx
location /cli/ {
    alias /var/www/pachim.sh/cli/;
    autoindex off;
}
```

### 3. GitHub repository variable

In `pachim/cli` → **Settings → Secrets and variables → Actions → Variables**:

| Variable | Value |
|----------|--------|
| `PACHIM_MIRROR_ENABLED` | `true` |
| `PACHIM_DEPLOY_PATH` | `/var/www/pachim.sh/cli` (optional, this is the default in script) |

### 4. GitHub secrets for SSH upload

| Secret | Value |
|--------|--------|
| `PACHIM_DEPLOY_HOST` | e.g. `pachim.sh` or server IP |
| `PACHIM_DEPLOY_USER` | e.g. `deploy` |
| `PACHIM_DEPLOY_KEY` | private SSH key (full PEM) |

The deploy user must be able to write to `PACHIM_DEPLOY_PATH`.

### 5. One-time: host install scripts manually

Until the first release runs, you can copy install scripts once:

```bash
scp install/install.sh install/install.ps1 user@server:/var/www/pachim.sh/cli/
```

After the first tagged release, CI updates them automatically.

## User installation

**Linux / macOS (mirror first, GitHub fallback):**

```bash
curl -fsSL https://pachim.sh/cli/install.sh | sh
```

**Windows:**

```powershell
irm https://pachim.sh/cli/install.ps1 | iex
```

## Local release test (without publishing)

```bash
go install github.com/goreleaser/goreleaser/v2@latest
goreleaser release --snapshot --clean
ls dist/
```
