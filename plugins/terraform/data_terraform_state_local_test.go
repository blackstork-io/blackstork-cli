package terraform

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics/diagtest"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
	"github.com/blackstork-io/fabric/internal/plugin/plugintest"
)

func Test_terraformStateLocalDataSchema(t *testing.T) {
	source := makeTerraformStateLocalDataSource()
	assert.Nil(t, source.Config)
	assert.NotNil(t, source.Args)
	assert.NotNil(t, source.DataFunc)
}

func Test_fetchTerraformStateLocalData_Call(t *testing.T) {
	type result struct {
		data  plugindata.Data
		diags diagnostics.Diag
	}
	tt := []struct {
		name        string
		path        string
		expectedRes plugindata.Data
		asserts     diagtest.Asserts
	}{
		{
			name: "notfound",
			path: filepath.Join("testdata", "notfound.tfstate"),
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryEquals("Failed to read terraform state"),
				diagtest.DetailContains("notfound.tfstate"),
			}},
		},
		{
			name: "empty_path",
			path: "",
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryEquals("Missing required attribute"),
				diagtest.DetailContains("path", "is required"),
			}},
		},
		{
			name: "valid",
			path: filepath.Join("testdata", "terraform.tfstate"),
			expectedRes: plugindata.Map{
				"version": plugindata.Number(1),
				"serial":  plugindata.Number(0),
				"modules": plugindata.List{
					plugindata.Map{
						"path": plugindata.List{
							plugindata.String("root"),
						},
						"outputs":   plugindata.Map{},
						"resources": plugindata.Map{},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			p := Plugin("1.2.3")
			args := plugintest.NewTestDecoder(t, p.DataSources["terraform_state_local"].Args)
			if tc.path != "" {
				args.SetAttr("path", cty.StringVal(tc.path))
			}
			val, fm, diags := args.DecodeDiagFiles()
			var (
				res  plugindata.Data
				diag diagnostics.Diag
			)
			if !diags.HasErrors() {
				res, diag = p.RetrieveData(context.Background(), "terraform_state_local", &plugin.RetrieveDataParams{
					Args: val,
				})
				diags.Extend(diag)
			}
			tc.asserts.AssertMatch(t, diags, fm)
			assert.Equal(t, tc.expectedRes, res)
		})
	}
}
