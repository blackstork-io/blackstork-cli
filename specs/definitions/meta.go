// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package definitions

import (
	"slices"

	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

type MetaBlock struct {
	Name        string   `hcl:"name,optional"`
	Description string   `hcl:"description,optional"`
	URL         string   `hcl:"url,optional"`
	License     string   `hcl:"license,optional"`
	Authors     []string `hcl:"authors,optional"`
	Tags        []string `hcl:"tags,optional"`
	UpdatedAt   string   `hcl:"updated_at,optional"`
	Version     string   `hcl:"version,optional"`

	// TODO: ?store def range defRange hcl.Range
}

func (m *MetaBlock) MatchesTags(requiredTags []string) bool {
	var tags []string
	if m != nil {
		tags = m.Tags
	}
	if len(tags) < len(requiredTags) {
		return false
	}
	for _, tag := range requiredTags {
		if !slices.Contains(tags, tag) {
			return false
		}
	}
	return true
}

func (m *MetaBlock) AsPluginData() plugindata.Map {
	tags := make(plugindata.List, len(m.Tags))
	authors := make(plugindata.List, len(m.Authors))
	for i, tag := range m.Tags {
		tags[i] = plugindata.String(tag)
	}
	for i, author := range m.Authors {
		authors[i] = plugindata.String(author)
	}
	return plugindata.Map{
		"name":        plugindata.String(m.Name),
		"description": plugindata.String(m.Description),
		"url":         plugindata.String(m.URL),
		"license":     plugindata.String(m.License),
		"authors":     authors,
		"tags":        tags,
		"updated_at":  plugindata.String(m.UpdatedAt),
		"version":     plugindata.String(m.Version),
	}
}
