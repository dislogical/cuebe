#!/usr/bin/env bash
# Copyright © 2025 Colden Cullen
# SPDX-License-Identifier: MIT

declare -a SRCPATHS=(
  "api/go"
)

for SRCPATH in "${SRCPATHS[@]}"; do
  # If there is no tag, use the short commit
  TAG=${1:-$(git rev-parse --short $(git rev-list -1 HEAD -- $SRCPATH))}

  go tool gomarkdoc go.bonk.build/$SRCPATH --repository.default-branch=$TAG
done
