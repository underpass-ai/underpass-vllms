#!/usr/bin/env bash
set -euo pipefail

SOURCE="${SOURCE:-/tmp/operator-qwen05-lora-v8.1.2-sft-v2}"
TARGET="${TARGET:-/var/lib/operator-adapters/v8.1.2-sft-v2-canonical}"
EXPECTED_SHA="${EXPECTED_SHA:-43186fa848c5f0e9d71915023f8f01c2341042de8aaf57b0c3c0c574a0f44379}"

if [ ! -d "$SOURCE" ]; then
  echo "source adapter directory does not exist: $SOURCE" >&2
  exit 1
fi

if [ ! -f "$SOURCE/adapter_model.safetensors" ]; then
  echo "source adapter_model.safetensors does not exist: $SOURCE/adapter_model.safetensors" >&2
  exit 1
fi

install -d "$TARGET"
cp -a "$SOURCE"/. "$TARGET"/

ACTUAL_SHA="$(sha256sum "$TARGET/adapter_model.safetensors" | cut -d' ' -f1)"
if [ "$ACTUAL_SHA" != "$EXPECTED_SHA" ]; then
  echo "SHA mismatch for $TARGET/adapter_model.safetensors" >&2
  echo "expected: $EXPECTED_SHA" >&2
  echo "actual:   $ACTUAL_SHA" >&2
  exit 1
fi

echo "Adapter installed at $TARGET"
echo "SHA verified: $ACTUAL_SHA"
