#!/usr/bin/env bash
set -euo pipefail

# Calculate next versions based on existing git tags
# Usage: ./calculate-next-versions.sh [evm_override] [sdk_override]
#
# Outputs (to stdout):
#   evm_version=v1.0.0-beta.X
#   sdk_version=v1.1.X
#
# If running in GitHub Actions, also writes to $GITHUB_OUTPUT

EVM_OVERRIDE="${1:-}"
SDK_OVERRIDE="${2:-}"

# Get latest EVM capability tag
LATEST_EVM_TAG=$(git tag -l "capabilities/blockchain/evm/v*" --sort=-v:refname | head -n1 || echo "")
echo "Latest EVM tag: ${LATEST_EVM_TAG}" >&2

if [ -n "$EVM_OVERRIDE" ]; then
  EVM_VERSION="$EVM_OVERRIDE"
elif [ -n "$LATEST_EVM_TAG" ]; then
  # Auto-increment: extract version and increment beta number
  # Format: capabilities/blockchain/evm/v1.0.0-beta.1 -> v1.0.0-beta.2
  LATEST_EVM_VERSION=${LATEST_EVM_TAG#capabilities/blockchain/evm/}
  
  if [[ "$LATEST_EVM_VERSION" =~ ^(v[0-9]+\.[0-9]+\.[0-9]+-beta\.)([0-9]+)$ ]]; then
    PREFIX="${BASH_REMATCH[1]}"
    BETA_NUM="${BASH_REMATCH[2]}"
    NEXT_BETA=$((BETA_NUM + 1))
    EVM_VERSION="${PREFIX}${NEXT_BETA}"
  else
    # Non-beta format, increment patch: v1.0.0 -> v1.0.1
    if [[ "$LATEST_EVM_VERSION" =~ ^(v[0-9]+\.[0-9]+\.)([0-9]+)$ ]]; then
      PREFIX="${BASH_REMATCH[1]}"
      PATCH="${BASH_REMATCH[2]}"
      NEXT_PATCH=$((PATCH + 1))
      EVM_VERSION="${PREFIX}${NEXT_PATCH}"
    else
      EVM_VERSION=""
      echo "Could not parse EVM version format: ${LATEST_EVM_VERSION}" >&2
    fi
  fi
else
  # No existing tag, start at v1.0.0-beta.1
  EVM_VERSION="v1.0.0-beta.1"
fi

echo "Next EVM version: ${EVM_VERSION}" >&2

# Get latest SDK tag (format: v1.1.1)
LATEST_SDK_TAG=$(git tag -l "v[0-9]*.[0-9]*.[0-9]*" --sort=-v:refname | grep -E "^v[0-9]+\.[0-9]+\.[0-9]+$" | head -n1 || echo "")
echo "Latest SDK tag: ${LATEST_SDK_TAG}" >&2

if [ -n "$SDK_OVERRIDE" ]; then
  SDK_VERSION="$SDK_OVERRIDE"
elif [ -n "$LATEST_SDK_TAG" ]; then
  # Auto-increment patch: v1.1.1 -> v1.1.2
  if [[ "$LATEST_SDK_TAG" =~ ^(v[0-9]+\.[0-9]+\.)([0-9]+)$ ]]; then
    PREFIX="${BASH_REMATCH[1]}"
    PATCH="${BASH_REMATCH[2]}"
    NEXT_PATCH=$((PATCH + 1))
    SDK_VERSION="${PREFIX}${NEXT_PATCH}"
  else
    SDK_VERSION=""
    echo "Could not parse SDK version format: ${LATEST_SDK_TAG}" >&2
  fi
else
  # No existing tag, start at v1.0.0
  SDK_VERSION="v1.0.0"
fi

echo "Next SDK version: ${SDK_VERSION}" >&2

# Output for consumption
echo "evm_version=${EVM_VERSION}"
echo "sdk_version=${SDK_VERSION}"

# If in GitHub Actions, write to GITHUB_OUTPUT
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "evm_version=${EVM_VERSION}" >> "$GITHUB_OUTPUT"
  echo "sdk_version=${SDK_VERSION}" >> "$GITHUB_OUTPUT"
fi

