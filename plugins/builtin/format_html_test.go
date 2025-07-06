package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

func Test_makeHTMLFormatter(t *testing.T) {
	schema := makeHTMLFormatter(nil, nil)
	assert.NotNil(t, schema.Doc)
	assert.Equal(t, "html", schema.Format)
	assert.Equal(t, "html", schema.FileExt)
	assert.NotNil(t, schema.Args)
	assert.NotNil(t, schema.FormatFunc)
}

func Test_formatHTML(t *testing.T) {
	schema := makeHTMLFormatter(nil, nil)

	titleMeta := plugindata.Map{
		"provider": plugindata.String("title"),
		"plugin":   plugindata.String("blackstork/builtin"),
	}

	params := &plugin.FormatParams{
		Args: dataspec.NewBlock([]string{"format"}, map[string]cty.Value{
			"page_title": cty.StringVal("Title {{.document.meta.name}}"),
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
