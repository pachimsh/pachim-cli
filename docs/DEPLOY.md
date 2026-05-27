# Release & S3 Mirror Deployment

## Overview

1. Push tag: `git tag v0.1.0 && git push origin v0.1.0`
2. GitHub Actions builds with GoReleaser → GitHub Release
3. Same files upload to S3 (for users without GitHub access)
4. Users install via `https://mirrors.pachim.app/cli/install.sh`

---

## Step 1 — Create S3 bucket

Use **Arvan Object Storage**, **MinIO**, or any S3-compatible provider.

Example bucket name: `pachim-mirrors`

Enable **public read** for objects under `cli/*` (bucket policy or public bucket — provider-specific).

Map custom domain **`mirrors.pachim.app`** to this bucket (CDN or provider settings).

### S3 layout after release

```
s3://pachim-mirrors/
├── cli/
│   ├── install.sh
│   ├── install.ps1
│   ├── latest.txt          ← e.g. v0.1.0
│   └── v0.1.0/
│       ├── pachim_linux_amd64.tar.gz
│       ├── pachim_windows_amd64.zip
│       └── checksums.txt
```

---

## Step 2 — Create access key (minimal permissions)

Create a key that can **only** write/read your CLI bucket (not other buckets).

Minimum permissions (conceptually):

- `s3:PutObject`, `s3:GetObject`, `s3:ListBucket`
- Resource: `arn:...:pachim-cli` and `arn:...:pachim-cli/cli/*`

Do **not** grant `s3:DeleteBucket` or full `s3:*`.

---

## Step 3 — Public download URL

Choose one:

**A) Direct S3/CDN URL** (if bucket is public):

```
https://YOUR-BUCKET-PUBLIC-URL/cli/latest.txt
```

**B) Custom domain (recommended):**

Public download base URL:

```
https://mirrors.pachim.app/cli/
```

CDN should point `mirrors.pachim.app` to bucket `pachim-mirrors` (root). Files live under the `cli/` prefix, so URLs include `/cli/`.

---

## Step 4 — GitHub: create repositories

| Repo | Purpose |
|------|---------|
| `pachimsh/pachim-cli` | Main code + Actions |
| `pachimsh/homebrew-tap` | Empty (GoReleaser fills) |
| `pachimsh/scoop-bucket` | Empty (GoReleaser fills) |

**Settings → Actions → General → Workflow permissions:** Read and write.

---

## Step 5 — GitHub Secrets

`pachimsh/pachim-cli` → **Settings → Secrets and variables → Actions → Secrets**

| Secret | Example |
|--------|---------|
| `GH_PAT` | GitHub PAT with `repo` (for Homebrew/Scoop) |
| `S3_ENDPOINT` | `https://s3.ir-thr-at1.arvanstorage.ir` |
| `S3_BUCKET` | `pachim-mirrors` |
| `S3_ACCESS_KEY_ID` | your access key |
| `S3_SECRET_ACCESS_KEY` | your secret key |
| `S3_REGION` | e.g. `ir-thr-at1` (if required by provider) |

---

## Step 6 — GitHub Variables

**Settings → Secrets and variables → Actions → Variables**

| Variable | Value |
|----------|--------|
| `S3_MIRROR_ENABLED` | `true` |
| `ENABLE_PACKAGE_MANAGERS` | `true` — only after `homebrew-tap` + `scoop-bucket` exist **and** `GH_PAT` is set |
| `S3_PREFIX` | `cli` (optional, default in script is `cli`) |
| `S3_PUBLIC_BASE_URL` | `https://mirrors.pachim.app/cli` (documentation only) |
| `S3_ACL` | `public-read` (only if your provider needs ACL on upload; Arvan often uses bucket policy instead) |
| `S3_FORCE_PATH_STYLE` | `true` (for MinIO; leave empty for Arvan) |

By default, GoReleaser **skips** Homebrew/Scoop so the first release only needs the built-in `GITHUB_TOKEN`. Enable package managers later with both `GH_PAT` and `ENABLE_PACKAGE_MANAGERS=true`.

---

## Step 7 — Push code and release

```bash
cd pachim-cli
git remote add origin https://github.com/pachimsh/pachim-cli.git   # first time only
git add .
git commit -m "Initial release pipeline with S3 mirror"
git push -u origin main

git tag v0.1.0
git push origin v0.1.0
```

Check **Actions** tab → workflow should be green.

Verify:

- GitHub → Releases → assets present
- S3 → `cli/v0.1.0/` contains binaries
- `cli/latest.txt` contains `v0.1.0`
- `curl https://mirrors.pachim.app/cli/latest.txt`

---

## Step 8 — User installation

```bash
curl -fsSL https://mirrors.pachim.app/cli/install.sh | sh
```

```powershell
irm https://mirrors.pachim.app/cli/install.ps1 | iex
```

Scripts try **mirror first**, then **GitHub** as fallback.

---

## Security notes

- Secrets stay in GitHub — never commit keys to git
- Use a dedicated bucket + scoped access key
- Rotate keys periodically
- Tag-only releases (`v*`) — workflow does not run on random PRs

---

## Troubleshooting — GoReleaser exit code 1

Open the failed **Actions** run → **Run GoReleaser** step and scroll to the last red lines.

| Log message | Fix |
|-------------|-----|
| `repository not found` / `404` on `pachimsh/cli` | `.goreleaser.yml` must use `name: pachim-cli` (same as GitHub repo name, not the Go module path) |
| `repository not found` / `404` on `homebrew-tap` or `scoop-bucket` | Create empty repos under `pachimsh`, or keep `--skip=brew,scoop` in the workflow |
| `resource not accessible by integration` | Default token cannot push to other repos — add `GH_PAT` with `repo` scope, or keep brew/scoop skipped |
| `git is dirty` | Commit all changes before tagging |
| `version does not start with v` | Tag must be `v0.1.0`, not `0.1.0` |
| Go version errors | `setup-go` uses `go.mod`; ensure that Go version is available on GitHub runners |

**Node.js 20 deprecation** in the workflow log is a warning only; it does not fail the job. The workflow sets `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=true`.

**`goreleaser check` exit code 2** means the config is valid but uses deprecated options (e.g. `brews` in GoReleaser 2.16). That is not a hard failure — do not fail CI on exit code 2.

---

## Local test (no publish)

```bash
goreleaser release --snapshot --clean
ls dist/
```
