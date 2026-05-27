#!/usr/bin/env bash
set -euo pipefail

VERSION="${GITHUB_REF_NAME:?GITHUB_REF_NAME is required}"
if [ -z "${PACHIM_DEPLOY_PATH:-}" ]; then
  DEPLOY_PATH="/var/www/pachim.sh/cli"
else
  DEPLOY_PATH="${PACHIM_DEPLOY_PATH}"
fi
HOST="${PACHIM_DEPLOY_HOST:?PACHIM_DEPLOY_HOST is required}"
USER="${PACHIM_DEPLOY_USER:?PACHIM_DEPLOY_USER is required}"

REMOTE="${USER}@${HOST}"
REMOTE_VERSION="${DEPLOY_PATH}/${VERSION}"

echo "Uploading release ${VERSION} to ${REMOTE}:${REMOTE_VERSION}"

ssh -o StrictHostKeyChecking=no "${REMOTE}" "mkdir -p '${REMOTE_VERSION}'"

rsync -avz --delete \
  dist/ \
  "${REMOTE}:${REMOTE_VERSION}/"

scp -o StrictHostKeyChecking=no \
  install/install.sh \
  install/install.ps1 \
  "${REMOTE}:${DEPLOY_PATH}/"

echo "${VERSION}" | ssh -o StrictHostKeyChecking=no "${REMOTE}" "cat > '${DEPLOY_PATH}/latest.txt'"

if [ -f dist/checksums.txt ]; then
  scp -o StrictHostKeyChecking=no \
    dist/checksums.txt \
    "${REMOTE}:${REMOTE_VERSION}/checksums.txt"
fi

echo "Mirror upload complete."
