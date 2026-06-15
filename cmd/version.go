// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package cmd

import (
	"fmt"
	"runtime/debug"
	"slices"
	"strings"
)

// Overridden by goreleaser.
var (
	version = ""
	builtBy = "golang"
)

func init() {
	if builtBy != "goreleaser" {
		version = fmt.Sprintf(
			"%s+builtBy.%s",
			versionFromBuildInfo(),
			builtBy,
		)
	}
	// Version needs to be set here to the command instead of where rootCmd is defined
	// because the version is set after the rootCmd is defined. Else, the version
	// will be empty and the command will not show the version.
	rootCmd.Version = version
}

func versionFromBuildInfo() (result string) {
	result = "v0.0.0-dev"
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		return result
	}
	if info.Main.Version != "(devel)" {
		result = strings.ToLower(info.Main.Version)
		if !strings.HasPrefix(result, "v") {
			result = "v" + result
		}
		return result
	}
	var meta []string
	// It's a dev version not built by goreleaser, add extra info
	dirtyIdx := slices.IndexFunc(info.Settings, func(s debug.BuildSetting) bool {
		return s.Key == "vcs.modified"
	})
	if dirtyIdx != -1 && info.Settings[dirtyIdx].Value == "true" {
		meta = append(meta, "dirty")
	}

	shaIdx := slices.IndexFunc(info.Settings, func(s debug.BuildSetting) bool {
		return s.Key == "vcs.revision"
	})
	if shaIdx != -1 {
		meta = append(meta, "rev", info.Settings[shaIdx].Value)
	}
	if len(meta) != 0 {
		result += "+" + strings.Join(meta, ".")
	}
	return result
}
