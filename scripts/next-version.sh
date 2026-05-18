#!/usr/bin/env bash
# Computes the next CalVer tag: YYYY.M.PATCH
# Finds the highest existing patch for the current year+month and increments it.
# Outputs nothing but the version string, suitable for: VERSION=$(bash scripts/next-version.sh)

set -euo pipefail

YEAR=$(date +%Y)
MONTH=$(date +%-m)   # no leading zero (GNU date; works on Linux)
PREFIX="${YEAR}.${MONTH}."

LAST=$(git tag --list "${PREFIX}*" | sort -t. -k3 -n | tail -1)

if [ -z "$LAST" ]; then
  echo "${YEAR}.${MONTH}.0"
else
  PATCH="${LAST##*.}"
  echo "${YEAR}.${MONTH}.$((PATCH + 1))"
fi
