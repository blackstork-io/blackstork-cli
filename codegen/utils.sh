#!/bin/bash

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
