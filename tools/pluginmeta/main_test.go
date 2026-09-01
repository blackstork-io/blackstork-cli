// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseConfigWithCurrentArchiveSchema(t *testing.T) {
	previousNamespace := namespace
	namespace = "blackstork"
	t.Cleanup(func() { namespace = previousNamespace })

	var cfg ReleaserConfig
	err := yaml.Unmarshal([]byte(`
builds:
  - id: plugin_example
    goos: [linux]
archives:
  - id: plugin_example
    formats: [tar.gz]
    ids: [plugin_example]
    name_template: "plugin_example_{{ .Os }}_{{ .Arch }}"
`), &cfg)
	require.NoError(t, err)

	meta, err := parseConfig(&cfg)
	require.NoError(t, err)
	require.Len(t, meta.Plugins, 1)
	require.Equal(t, "blackstork/example", meta.Plugins[0].Name)
	require.Len(t, meta.Plugins[0].Archives, 3)
	require.Equal(t, "plugin_example_linux_amd64.tar.gz", meta.Plugins[0].Archives[0].Filename)
}
