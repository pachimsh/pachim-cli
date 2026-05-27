#!/usr/bin/env bash
set -euo pipefail

VERSION="${GITHUB_REF_NAME:?GITHUB_REF_NAME is required}"
BUCKET="${S3_BUCKET:?S3_BUCKET is required}"
ENDPOINT="${S3_ENDPOINT:?S3_ENDPOINT is required}"
PREFIX="${S3_PREFIX:-cli}"

AWS_ARGS=(--endpoint-url "${ENDPOINT}")

if [ -n "${S3_REGION:-}" ]; then
  export AWS_DEFAULT_REGION="${S3_REGION}"
fi

if [ "${S3_FORCE_PATH_STYLE:-false}" = "true" ]; then
  AWS_ARGS+=(--cli-connect-timeout 60)
  export AWS_S3_FORCE_PATH_STYLE=true
fi

ACL_ARGS=()
if [ -n "${S3_ACL:-}" ]; then
  ACL_ARGS=(--acl "${S3_ACL}")
fi

echo "Uploading release ${VERSION} to s3://${BUCKET}/${PREFIX}/${VERSION}/"

aws s3 sync dist/ "s3://${BUCKET}/${PREFIX}/${VERSION}/" \
  "${AWS_ARGS[@]}" \
  "${ACL_ARGS[@]}" \
  --only-show-errors

aws s3 cp install/install.sh "s3://${BUCKET}/${PREFIX}/install.sh" \
  "${AWS_ARGS[@]}" \
  "${ACL_ARGS[@]}" \
  --content-type "text/plain"

aws s3 cp install/install.ps1 "s3://${BUCKET}/${PREFIX}/install.ps1" \
  "${AWS_ARGS[@]}" \
  "${ACL_ARGS[@]}" \
  --content-type "text/plain"

echo -n "${VERSION}" | aws s3 cp - "s3://${BUCKET}/${PREFIX}/latest.txt" \
  "${AWS_ARGS[@]}" \
  "${ACL_ARGS[@]}" \
  --content-type "text/plain"

if [ -f dist/checksums.txt ]; then
  aws s3 cp dist/checksums.txt "s3://${BUCKET}/${PREFIX}/${VERSION}/checksums.txt" \
    "${AWS_ARGS[@]}" \
    "${ACL_ARGS[@]}"
fi

echo "S3 upload complete."
echo "Public base URL (if configured): ${S3_PUBLIC_BASE_URL:-not set}"
