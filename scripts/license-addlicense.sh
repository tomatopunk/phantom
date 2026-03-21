#!/usr/bin/env bash
# Shared addlicense invocation (Apache-2.0 + SPDX). Used by Makefile and CI.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
COPYRIGHT="${LICENSE_COPYRIGHT:-The Phantom Authors}"
YEAR="${LICENSE_YEAR:-$(date +%Y)}"
exec go run github.com/google/addlicense@v1.1.1 \
  -l apache \
  -c "$COPYRIGHT" \
  -s \
  -y "$YEAR" \
  -ignore '**/node_modules/**' \
  -ignore '**/dist/**' \
  -ignore 'target/**' \
  -ignore '**/target/**' \
  -ignore '**/.git/**' \
  -ignore '**/*.pb.go' \
  -ignore '**/*.pb.gw.go' \
  -ignore '**/*.skel.h' \
  -ignore '**/gen/**' \
  -ignore '**/src-tauri/icons/**' \
  -ignore '**/Cargo.lock' \
  -ignore '**/package-lock.json' \
  -ignore 'LICENSE' \
  -ignore '**/*.md' \
  -ignore '**/*.png' \
  -ignore '**/*.ico' \
  -ignore '**/*.json' \
  -ignore '**/*.svg' \
  -ignore '**/*.webp' \
  -ignore '**/bpf/**/*.o' \
  -ignore '**/bpf/**/*.bpf.o' \
  "$@"
