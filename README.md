# Pachim CLI

A command-line tool for deploying projects to servers managed by [Pachim](https://pachim.sh).

## Installation

### Quick Install (Recommended)

**Linux / macOS:**

```bash
curl -fsSL https://mirrors.pachim.app/cli/install.sh | sh
```

Install scripts try **mirrors.pachim.app** first, then fall back to **GitHub** if the mirror is unavailable.

**Windows (PowerShell):**

```powershell
irm https://mirrors.pachim.app/cli/install.ps1 | iex
```

### Package Managers

**Homebrew (macOS / Linux):**

```bash
brew install pachimsh/homebrew-tap/pachim
```

**Scoop (Windows):**

```powershell
scoop bucket add pachimsh https://github.com/pachimsh/scoop-bucket
scoop install pachim
```

### Manual Download

Download the latest binary from [GitHub Releases](https://github.com/pachimsh/cli/releases) for your platform.

### Build from Source

```bash
go install github.com/pachimsh/cli@latest
```

Or clone and build:

```bash
git clone https://github.com/pachimsh/cli.git
cd cli
go build -o pachim .
```

## Usage

### 1. Login

```bash
pachim login
```

You'll be prompted for your email and password. The token is stored securely in `~/.pachim/profiles/default.json`.

### 2. Initialize a project

```bash
pachim init
```

This links the current directory to one or more Pachim sites. A `.pachim.json` file is created in the project root.

You can link multiple sites (e.g., staging and production) and switch between them using the `--site` flag.

### 3. Deploy

```bash
pachim push
```

This packages your project (using git-tracked files or respecting common ignore patterns), uploads it to Pachim, and monitors the deployment status.

To deploy to a specific site:

```bash
pachim push --site staging
```

### Other commands

```bash
pachim sites       # List all your sites
pachim whoami      # Show logged-in user
pachim profiles    # List all profiles
pachim logout      # Log out and remove stored credentials
```

## Multiple Profiles

You can use multiple Pachim accounts simultaneously:

```bash
# Login with different profiles
pachim --profile work login
pachim --profile personal login

# Deploy using a specific profile
pachim --profile work push

# Or set via environment variable
export PACHIM_PROFILE=work
pachim push
```

## Configuration

### Credentials

Stored at `~/.pachim/profiles/<name>.json` (file permissions: 0600). Never commit this file.

### Project config

`.pachim.json` in the project root:

```json
{
  "default": "example.com",
  "sites": {
    "example.com": {
      "site_id": "uuid-here",
      "domain": "example.com"
    },
    "staging": {
      "site_id": "uuid-staging",
      "domain": "staging.example.com"
    }
  }
}
```

## Development

### Custom API URL

For local development, override the API URL:

```bash
# Via environment variable
export PACHIM_API_URL=http://localhost:8000
pachim login

# Via flag
pachim --api-url http://localhost:8000 login

# Via build-time injection
go build -ldflags "-X main.apiBaseURL=http://localhost:8000" -o pachim .
```

### Build

```bash
# Build for current OS
go build -o pachim .

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o pachim-linux .
GOOS=darwin GOARCH=amd64 go build -o pachim-mac .
GOOS=windows GOARCH=amd64 go build -o pachim.exe .
```

### Release

Releases are automated with GoReleaser and GitHub Actions. To create a new release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This will automatically build binaries for all platforms and publish to GitHub Releases, Homebrew, and Scoop.

For mirror deployment to `pachim.sh` (recommended for users in Iran), see [docs/DEPLOY.md](docs/DEPLOY.md).
