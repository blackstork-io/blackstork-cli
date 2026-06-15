#!/bin/bash
# Copyright 2026 BlackStork BV
#
# Use of this software is governed by the Business Source License included in the
# file LICENSE and at www.mariadb.com/bsl11.
#
# As of the Change Date specified in that file, in accordance with the Business
# Source License, use of this software will be governed by the Apache License,
# Version 2.0, included in the file .licenses/APACHE-2.0.txt.


# Collection of tools
function is_ci() {
    [ -n "$CI" ] && [ -n "$GITHUB_ACTIONS" ]
}

function install_tool() {
    local binary="$1"
    local path="$2"
    local version="$3"

    if $binary --version 2> /dev/null | grep -q "$version"; then
        # binary is already installed and has the correct version
        return
    fi

    if is_ci; then
        go install $path@v$version &
        return
    fi
    # avoid installing the binary into global scope
    # (perhaps developer has another version of the binary or does not want to install it)
    eval "function $binary() { go run \"$path@v$version\" \"\$@\"; }"
}
