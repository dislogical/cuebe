#!/usr/bin/env bash
# Copyright © 2025 Colden Cullen
# SPDX-License-Identifier: MIT

set -e

# Assumes cwd is the project root
# Incoming variables:
# CZ_PRE_IS_INITIAL           True when this is the initial release, False otherwise
# CZ_PRE_CURRENT_VERSION      Current version, before the bump
# CZ_PRE_CURRENT_TAG_VERSION  Current version tag, before the bump
# CZ_PRE_NEW_VERSION          New version, after the bump
# CZ_PRE_NEW_TAG_VERSION      New version tag, after the bump
# CZ_PRE_MESSAGE              Commit message of the bump
# CZ_PRE_INCREMENT            Whether this is a MAJOR, MINOR or PATH release
# CZ_PRE_CHANGELOG_FILE_NAME  Path to the changelog file, if available

# Generate the API docs
./scripts/gomarkdoc.sh $CZ_PRE_NEW_TAG_VERSION

# Publish the docs via mike
mike deploy --push --update-aliases $CZ_PRE_NEW_TAG_VERSION latest
