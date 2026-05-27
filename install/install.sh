#!/bin/sh
set -e

REPO="pachimsh/pachim-cli"
BINARY="pachim"
INSTALL_DIR="/usr/local/bin"
MIRROR_BASE="https://mirrors.pachim.app/cli"

main() {
    OS=$(detect_os)
    ARCH=$(detect_arch)

    echo "Detected: ${OS}/${ARCH}"

    LATEST=$(get_latest_version)
    echo "Latest version: ${LATEST}"

    FILENAME="${BINARY}_${OS}_${ARCH}.tar.gz"

    TMPDIR=$(mktemp -d)
    trap "rm -rf ${TMPDIR}" EXIT

    if ! download_archive "${LATEST}" "${FILENAME}" "${TMPDIR}"; then
        echo "Error: failed to download ${FILENAME}" >&2
        exit 1
    fi

    echo "Extracting..."
    tar xzf "${TMPDIR}/${FILENAME}" -C "${TMPDIR}"

    echo "Installing to ${INSTALL_DIR}..."
    if [ -w "${INSTALL_DIR}" ]; then
        mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    else
        sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    fi
    chmod +x "${INSTALL_DIR}/${BINARY}"

    echo ""
    echo "✓ pachim ${LATEST} installed successfully!"
    echo "  Run 'pachim --help' to get started."
}

download_archive() {
    version="$1"
    filename="$2"
    tmpdir="$3"

    mirror_url="${MIRROR_BASE}/${version}/${filename}"
    if download_file "${mirror_url}" "${tmpdir}/${filename}"; then
        echo "Downloaded from pachim.sh mirror."
        return 0
    fi

    echo "Mirror unavailable, trying GitHub..."
    github_url="https://github.com/${REPO}/releases/download/${version}/${filename}"
    if download_file "${github_url}" "${tmpdir}/${filename}"; then
        echo "Downloaded from GitHub."
        return 0
    fi

    return 1
}

download_file() {
    url="$1"
    dest="$2"

    echo "Downloading ${url}..."
    if command -v curl > /dev/null 2>&1; then
        curl -fsSL "${url}" -o "${dest}"
        return $?
    fi
    if command -v wget > /dev/null 2>&1; then
        wget -q "${url}" -O "${dest}"
        return $?
    fi

    echo "Error: curl or wget is required" >&2
    return 1
}

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *)              echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
    esac
}

get_latest_version() {
    if command -v curl > /dev/null 2>&1; then
        version=$(curl -fsSL "${MIRROR_BASE}/latest.txt" 2>/dev/null | tr -d '[:space:]')
        if [ -n "${version}" ]; then
            echo "${version}"
            return 0
        fi
        curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
        return 0
    fi
    if command -v wget > /dev/null 2>&1; then
        version=$(wget -qO- "${MIRROR_BASE}/latest.txt" 2>/dev/null | tr -d '[:space:]')
        if [ -n "${version}" ]; then
            echo "${version}"
            return 0
        fi
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
        return 0
    fi

    echo "Error: curl or wget is required" >&2
    exit 1
}

main
