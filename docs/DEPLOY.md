# Release & S3 Mirror Deployment

## Overview

1. Push tag: `git tag v0.1.0 && git push origin v0.1.0`
2. GitHub Actions builds with GoReleaser → GitHub Release
3. Same files upload to S3 (for users without GitHub access)
4. Users install via `https://pachim.sh/cli/install.sh` (or your S3/CDN URL)

---

## Step 1 — Create S3 bucket

Use **Arvan Object Storage**, **MinIO**, or any S3-compatible provider.

Example bucket name: `pachim-cli`

Enable **public read** for objects under `cli/*` (bucket policy or public bucket — provider-specific).

### S3 layout after release

```
s3://pachim-cli/
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

Point `https://pachim.sh/cli/` (or `https://dl.pachim.sh/cli/`) to your bucket via:

- Provider CDN / custom domain (Arvan)
- Or Nginx reverse proxy on your server

Install scripts default to `https://pachim.sh/cli` — keep that URL working.

---

## Step 4 — GitHub: create repositories

| Repo | Purpose |
|------|---------|
| `pachimsh/cli` | Main code + Actions |
| `pachimsh/homebrew-tap` | Empty (GoReleaser fills) |
| `pachimsh/scoop-bucket` | Empty (GoReleaser fills) |

**Settings → Actions → General → Workflow permissions:** Read and write.

---

## Step 5 — GitHub Secrets

`pachimsh/cli` → **Settings → Secrets and variables → Actions → Secrets**

| Secret | Example |
|--------|---------|
| `GH_PAT` | GitHub PAT with `repo` (for Homebrew/Scoop) |
| `S3_ENDPOINT` | `https://s3.ir-thr-at1.arvanstorage.ir` |
| `S3_BUCKET` | `pachim-cli` |
| `S3_ACCESS_KEY_ID` | your access key |
| `S3_SECRET_ACCESS_KEY` | your secret key |
| `S3_REGION` | e.g. `ir-thr-at1` (if required by provider) |

---

## Step 6 — GitHub Variables

**Settings → Secrets and variables → Actions → Variables**

| Variable | Value |
|----------|--------|
| `S3_MIRROR_ENABLED` | `true` |
| `S3_PREFIX` | `cli` (optional, default in script is `cli`) |
| `S3_PUBLIC_BASE_URL` | `https://pachim.sh/cli` (documentation only) |
| `S3_ACL` | `public-read` (only if your provider needs ACL on upload; Arvan often uses bucket policy instead) |
| `S3_FORCE_PATH_STYLE` | `true` (for MinIO; leave empty for Arvan) |

---

## Step 7 — Push code and release

```bash
cd pachim-cli
git remote add origin https://github.com/pachimsh/cli.git   # first time only
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
- `curl https://pachim.sh/cli/latest.txt` (or your public URL)

---

## Step 8 — User installation

```bash
curl -fsSL https://pachim.sh/cli/install.sh | sh
```

```powershell
irm https://pachim.sh/cli/install.ps1 | iex
```

Scripts try **mirror first**, then **GitHub** as fallback.

---

## Security notes

- Secrets stay in GitHub — never commit keys to git
- Use a dedicated bucket + scoped access key
- Rotate keys periodically
- Tag-only releases (`v*`) — workflow does not run on random PRs

---

## Local test (no publish)

```bash
goreleaser release --snapshot --clean
ls dist/
```
