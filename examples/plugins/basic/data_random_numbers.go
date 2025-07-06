package basic

import (
	"context"
	"math/rand"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

// makeRandomNumbersDataSource creates a new data source for generating random numbers
func makeRandomNumbersDataSource() *plugin.DataSource {
	return &plugin.DataSource{
		// Config is optional, we can define the schema for the config that is reusable for this data source
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:       "min",
					Type:       cty.Number,
					Doc:        `Lower bound (inclusive)`,
					DefaultVal: cty.NumberIntVal(0),
				},
				{
					Name:       "max",
					Type:       cty.Number,
					Doc:        `Upper bound (inclusive)`,
					DefaultVal: cty.NumberIntVal(100),
				},
			},
		},
		// We define the schema for the arguments
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:         "length",
					Constraints:  constraint.Integer | constraint.RequiredNonNull,
					Type:         cty.Number,
					ExampleVal:   cty.NumberIntVal(10),
					MinInclusive: cty.NumberIntVal(1),
				},
			},
		},
		// Optional: We can also define the schema for the config
		DataFunc: fetchRandomNumbers,
	}
}

func fetchRandomNumbers(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
	min := params.Config.GetAttrVal("min")
	max := params.Config.GetAttrVal("max")

	// validating the arguments
	length := params.Args.GetAttrVal("length")
	if min.GreaterThan(max).True() {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse config",
			Detail:   "min is greater than max",
		}}
	}

	lengthInt, _ := length.AsBigFloat().Int64()
	minInt, _ := min.AsBigFloat().Int64()
	maxInt, _ := max.AsBigFloat().Int64()

	data := make(plugindata.List, lengthInt)
	for i := int64(0); i < lengthInt; i++ {
		n := rand.Int63() % (maxInt - minInt + 1) //nolint:G404 // weak rng is ok here
		data[i] = plugindata.Number(n + minInt)
	}
	return data, nil
}
