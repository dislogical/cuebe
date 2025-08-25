#!/usr/bin/env bash
# Copyright © 2025 Colden Cullen
# SPDX-License-Identifier: MIT

set -e

if [ "$#" -ne 2 ]; then
  echo api_version_bump.sh requires 2 arguments
  exit 1
fi

echo MAJOR VERSION
echo Updating API revision

CZ_PRE_CURRENT_MAJOR=$1
CZ_PRE_NEW_MAJOR=$2

echo "Moving from $CZ_PRE_CURRENT_MAJOR to $CZ_PRE_NEW_MAJOR"

# Move api folders
find ./api -type d -name "$CZ_PRE_CURRENT_MAJOR" \
  -execdir mv $CZ_PRE_CURRENT_MAJOR $CZ_PRE_NEW_MAJOR ';' \
    2>/dev/null

# Regenerate protos
go tool buf generate

# Rename any packages/imports
find ./api/ ./pkg/ -type f \
  -exec sed -r -i "s|bonk([/.]?)$CZ_PRE_CURRENT_MAJOR|bonk\1$CZ_PRE_NEW_MAJOR|g" {} ';' \
  -exec sed -r -i 's|ProtocolVersion:(\s*)$CZ_PRE_CURRENT_MAJOR|ProtocolVersion:\1$CZ_PRE_NEW_MAJOR|g' {} ';'
