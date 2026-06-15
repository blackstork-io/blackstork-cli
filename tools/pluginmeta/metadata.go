// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package main

type Metadata struct {
	Plugins []*PluginMetadata `json:"plugins"`
}

type PluginMetadata struct {
	Name     string                   `json:"name"`
	Version  string                   `json:"version"`
	Archives []*PluginArchiveMetadata `json:"archives"`
}

type PluginArchiveMetadata struct {
	Filename       string `json:"filename"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	BinaryChecksum string `json:"binary_checksum"`
}
