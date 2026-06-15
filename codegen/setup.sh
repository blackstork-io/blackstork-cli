#!/bin/bash
# Copyright 2026 BlackStork BV
#
# Use of this software is governed by the Business Source License included in the
# file LICENSE and at www.mariadb.com/bsl11.
#
# As of the Change Date specified in that file, in accordance with the Business
# Source License, use of this software will be governed by the Apache License,
# Version 2.0, included in the file .licenses/APACHE-2.0.txt.

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
