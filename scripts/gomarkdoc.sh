#!/usr/bin/env bash
# Copyright © 2025 Colden Cullen
# SPDX-License-Identifier: MIT

SRCPATH="${1:-.}"

TAG=$(git tag --points-at HEAD)

# If there is no tag, use the short commit
if [ -z "$TAG" ]; then
  TAG=$(git rev-parse --short $(git rev-list -1 HEAD -- $SRCPATH))
fi

go tool gomarkdoc go.bonk.build/$SRCPATH --repository.default-branch=$TAG
