// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package main

type ReleaserConfig struct {
	Builds   []ReleaserBuild   `yaml:"builds"`
	Archives []ReleaserArchive `yaml:"archives"`
}

type ReleaserBuild struct {
	ID   string   `yaml:"id"`
	GOOS []string `yaml:"goos"`
}

type ReleaserArchive struct {
	ID           string   `yaml:"id"`
	Formats      []string `yaml:"formats"`
	IDs          []string `yaml:"ids"`
	NameTemplate string   `yaml:"name_template"`
}

type ReleaserFormatOverride struct {
	GOOS    string   `yaml:"goos"`
	Formats []string `yaml:"formats"`
}
