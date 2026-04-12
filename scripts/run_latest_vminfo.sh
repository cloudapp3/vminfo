#!/usr/bin/env bash
set -euo pipefail

PROJECT_PATH="${PROJECT_PATH:-cloudgithub/vminfo}"
PROJECT_PATH_ENC="${PROJECT_PATH//\//%2F}"
REF="${REF:-main}"
JOB="${JOB:-build-linux-amd64}"
ARTIFACT_PATH="${ARTIFACT_PATH:-dist/vminfo-linux-amd64}"
DEFAULT_ARGS=(summary)

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

need_cmd curl
need_cmd mktemp
need_cmd chmod
need_cmd uname

OS="$(uname -s)"
ARCH="$(uname -m)"

if [[ "$OS" != "Linux" ]]; then
  echo "only Linux is supported by this helper, current OS: $OS" >&2
  exit 1
fi

if [[ "$ARCH" != "x86_64" && "$ARCH" != "amd64" ]]; then
  echo "only amd64/x86_64 artifact is supported, current arch: $ARCH" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

BIN_PATH="$TMP_DIR/vminfo"
URL="https://gitlab.com/api/v4/projects/${PROJECT_PATH_ENC}/jobs/artifacts/${REF}/raw/${ARTIFACT_PATH}?job=${JOB}"

CURL_ARGS=(
  --fail
  --location
  --silent
  --show-error
  --output "$BIN_PATH"
)

if [[ -n "${GITLAB_TOKEN:-}" ]]; then
  CURL_ARGS=(--header "PRIVATE-TOKEN: ${GITLAB_TOKEN}" "${CURL_ARGS[@]}")
fi

curl "${CURL_ARGS[@]}" "$URL"
chmod +x "$BIN_PATH"

if [[ $# -eq 0 ]]; then
  set -- "${DEFAULT_ARGS[@]}"
fi

exec "$BIN_PATH" "$@"
