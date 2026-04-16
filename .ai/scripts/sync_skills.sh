#!/usr/bin/env bash
set -euo pipefail

# Keep runtime skills mirror synced from canonical repo skills.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SRC_DIR="$ROOT_DIR/.ai/skills"
DST_DIR="$ROOT_DIR/.agents/skills"

if [[ ! -d "$SRC_DIR" ]]; then
  echo "source directory not found: $SRC_DIR" >&2
  exit 1
fi

mkdir -p "$DST_DIR"

# Remove old mirrored skills to avoid stale entries.
find "$DST_DIR" -mindepth 1 -maxdepth 1 -type d -exec rm -rf {} +

# Copy canonical skills to runtime mirror.
cp -R "$SRC_DIR"/. "$DST_DIR"/

echo "skills synced: $SRC_DIR -> $DST_DIR"
