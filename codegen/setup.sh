#!/bin/bash
# Make codegen tools available
set -e
source "$(dirname "${BASH_SOURCE[0]:-$0}")/utils.sh"
# Codegen tools
install_tool mockery github.com/vektra/mockery/v2 "2.52.2"
install_tool buf github.com/bufbuild/buf/cmd/buf "1.52.1"
# Formatting tools
install_tool gci github.com/daixiang0/gci "0.13.6"
install_tool gofumpt mvdan.cc/gofumpt "0.8.0"
wait
