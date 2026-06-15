#!/bin/bash
# Copyright 2026 BlackStork BV
#
# Use of this software is governed by the Business Source License included in the
# file LICENSE and at www.mariadb.com/bsl11.
#
# As of the Change Date specified in that file, in accordance with the Business
# Source License, use of this software will be governed by the Apache License,
# Version 2.0, included in the file .licenses/APACHE-2.0.txt.

# Removes all generated files

cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.."

grep -R --with-filename --files-with-matches --no-messages --include "*.go" -E -e '^// Code generated .* DO NOT EDIT.$' . | xargs rm
find ./mocks -type d -empty -exec rmdir {} +

find ./docs/plugins -mindepth 1 \( -maxdepth 1 -type d -o -not -name "*.md" \) -exec rm -rf {} +
