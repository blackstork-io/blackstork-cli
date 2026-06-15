// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package builtin

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPluginSchema(t *testing.T) {
	schema := Plugin("1.2.3", slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	assert.Equal(t, "blackstork/builtin", schema.Name)
	assert.Equal(t, "1.2.3", schema.Version)
	assert.NotNil(t, schema.DataSources["csv"])
	assert.NotNil(t, schema.DataSources["txt"])
	assert.NotNil(t, schema.DataSources["json"])
	assert.NotNil(t, schema.DataSources["rss"])
	// Content Providers
	assert.NotNil(t, schema.ContentProviders["toc"])
	assert.NotNil(t, schema.ContentProviders["text"])
	assert.NotNil(t, schema.ContentProviders["title"])
	assert.NotNil(t, schema.ContentProviders["code"])
	assert.NotNil(t, schema.ContentProviders["blockquote"])
	assert.NotNil(t, schema.ContentProviders["image"])
	assert.NotNil(t, schema.ContentProviders["list"])
	assert.NotNil(t, schema.ContentProviders["table"])
	assert.NotNil(t, schema.ContentProviders["frontmatter"])
	// Publishers
	assert.NotNil(t, schema.Publishers["local_file"])
	assert.NotNil(t, schema.Publishers["hub"])
}
