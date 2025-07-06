package pluginapiv1

import (
	"fmt"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
)

func encodeAttrSpec(src *dataspec.AttrSpec) (*AttrSpec, diagnostics.Diag) {
	ty, diags := encodeCtyType(src.Type)
	dv, diag := encodeCtyValue(src.DefaultVal)
	diags.Extend(diag)
	ev, diag := encodeCtyValue(src.ExampleVal)
	diags.Extend(diag)
	min, diag := encodeCtyValue(src.MinInclusive)
	diags.Extend(diag)
	max, diag := encodeCtyValue(src.MaxInclusive)
	diags.Extend(diag)
	return &AttrSpec{
		Name:         src.Name,
		Type:         ty,
		DefaultVal:   dv,
		ExampleVal:   ev,
		Doc:          src.Doc,
		Constraints:  uint32(src.Constraints),
		OneOf:        utils.FnMapDiags(&diags, src.OneOf, encodeCtyValue),
		MinInclusive: min,
		MaxInclusive: max,
		Deprecated:   src.Deprecated,
		Secret:       src.Secret,
	}, diags
}

func encodeRootSpec(src *dataspec.RootSpec) (*BlockSpec, diagnostics.Diag) {
	if src == nil {
		return nil, nil
	}
	return encodeBlockSpec(src.BlockSpec())
}

func encodeBlockSpec(src *dataspec.BlockSpec) (*BlockSpec, diagnostics.Diag) {
	var diags diagnostics.Diag
	matchers := make([]*BlockSpec_NameMatcher, len(src.Header))
	for i, s := range src.Header {
		switch sT := s.(type) {
		case dataspec.ExactMatcher:
			matchers[i] = &BlockSpec_NameMatcher{
				Matcher: &BlockSpec_NameMatcher_Exact_{
					Exact: &BlockSpec_NameMatcher_Exact{
						Matches: sT,
					},
				},
			}
		default:
			diags.Add(
				"Unknown NameMatcher type",
				fmt.Sprintf("Type %T is not supported", sT),
			)
		}
	}
	return &BlockSpec{
		HeadersSpec:                matchers,
		Required:                   src.Required,
		Repeatable:                 src.Repeatable,
		Doc:                        src.Doc,
		AttrSpecs:                  utils.FnMapDiags(&diags, src.Attrs, encodeAttrSpec),
		BlockSpecs:                 utils.FnMapDiags(&diags, src.Blocks, encodeBlockSpec),
		AllowUnspecifiedBlocks:     src.AllowUnspecifiedBlocks,
		AllowUnspecifiedAttributes: src.AllowUnspecifiedAttributes,
	}, diags
}

func encodeAttr(src *dataspec.Attr) (*Attr, diagnostics.Diag) {
	val, diags := encodeCtyValue(src.Value)
	return &Attr{
		Name:       src.Name,
		NameRange:  encodeRange(&src.NameRange),
		Value:      val,
		ValueRange: encodeRange(&src.ValueRange),
		Secret:     src.Secret,
	}, diags
}

func encodeBlock(src *dataspec.Block) (*Block, diagnostics.Diag) {
	var diags diagnostics.Diag
	return &Block{
		Header:        src.Header,
		HeaderRanges:  utils.FnMap(src.HeaderRanges, encodeRangeVal),
		Blocks:        utils.FnMapDiags(&diags, src.Blocks, encodeBlock),
		Attributes:    utils.MapMapDiags(&diags, src.Attrs, encodeAttr),
		ContentsRange: encodeRange(&src.ContentsRange),
	}, diags
}
