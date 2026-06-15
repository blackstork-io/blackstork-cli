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
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

func Test_makeLocalFilePublisher(t *testing.T) {
	schema := makeLocalFilePublisher(nil, nil)
	assert.NotNil(t, schema.Doc)
	assert.NotNil(t, schema.Tags)
	assert.NotNil(t, schema.Args)
	assert.Equal(t, []string{"md", "pdf", "html"}, schema.Formats)
	assert.NotNil(t, schema.PublishFunc)
}

func Test_publishLocalFileMD(t *testing.T) {
	schema := makeLocalFilePublisher(nil, nil)
	dir := t.TempDir()
	titleMeta := plugindata.Map{
		"provider": plugindata.String("title"),
		"plugin":   plugindata.String("blackstork/builtin"),
	}
	params := &plugin.PublishParams{
		Format: "md",
		Args: dataspec.NewBlock([]string{"local_file"}, map[string]cty.Value{
			"path": cty.StringVal(filepath.Join(dir, "{{.document.meta.name}}.{{.format}}")),
		}),
		DataContext: plugindata.Map{
			"document": plugindata.Map{
				"meta": plugindata.Map{
					"name": plugindata.String("test_document"),
				},
				"content": plugindata.Map{
					"type": plugindata.String("section"),
					"children": plugindata.List{
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("# Header 1"),
							"meta":     titleMeta,
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("Lorem ipsum dolor sit amet, consectetur adipiscing elit."),
						},
					},
				},
			},
		},
	}
	diags := schema.Execute(context.Background(), params)
	require.Empty(t, diags)
	bytes, err := os.ReadFile(filepath.Join(dir, "test_document.md"))
	require.NoError(t, err)
	got := string(bytes)
	assert.Equal(t, got, "# Header 1\n\nLorem ipsum dolor sit amet, consectetur adipiscing elit.")
}

func Test_publishLocalFileHTML(t *testing.T) {
	schema := makeLocalFilePublisher(nil, nil)
	dir := t.TempDir()
	titleMeta := plugindata.Map{
		"provider": plugindata.String("title"),
		"plugin":   plugindata.String("blackstork/builtin"),
	}
	params := &plugin.PublishParams{
		Format: "html",
		Args: dataspec.NewBlock([]string{"local_file"}, map[string]cty.Value{
			"path": cty.StringVal(filepath.Join(dir, "{{.document.meta.name}}.{{.format}}")),
		}),
		DataContext: plugindata.Map{
			"document": plugindata.Map{
				"meta": plugindata.Map{
					"name": plugindata.String("test_document"),
				},
				"content": plugindata.Map{
					"type": plugindata.String("section"),
					"children": plugindata.List{
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("# Header 1"),
							"meta":     titleMeta,
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("Lorem ipsum dolor sit amet, consectetur adipiscing elit."),
						},
					},
				},
			},
		},
	}
	diags := schema.Execute(context.Background(), params)
	require.Empty(t, diags)
	bytes, err := os.ReadFile(filepath.Join(dir, "test_document.html"))
	require.NoError(t, err)
	got := string(bytes)
	assert.Contains(t, got, "<h1 id=\"header-1\">Header 1</h1>")
	assert.Contains(t, got, "<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit.</p>")
}

func Test_publishLocalFile_invalidPath(t *testing.T) {
	schema := makeLocalFilePublisher(nil, nil)
	params := &plugin.PublishParams{
		Format: "md",
		Args: dataspec.NewBlock([]string{"local_file"}, map[string]cty.Value{
			"path": cty.StringVal(""),
		}),
		DataContext: plugindata.Map{
			"document": plugindata.Map{
				"content": plugindata.Map{
					"type": plugindata.String("section"),
					"children": plugindata.List{
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("# Header 1"),
						},
					},
				},
			},
		},
	}
	diags := schema.Execute(context.Background(), params)
	require.NotEmpty(t, diags)
}
