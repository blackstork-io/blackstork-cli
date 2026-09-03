// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package parser

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/require"

	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

func TestRefBlocksInheritMeta(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		kind   string
		runner string
	}{
		{kind: definitions.BlockKindData, runner: "test"},
		{kind: definitions.BlockKindContent, runner: "test"},
		{kind: definitions.BlockKindFormat, runner: "test"},
		{kind: definitions.BlockKindPublish, runner: "test"},
		{kind: definitions.BlockKindSection},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.kind, func(t *testing.T) {
			t.Parallel()

			labels := "\"base\""
			base := fmt.Sprintf("%s.base", tc.kind)
			if tc.runner != "" {
				labels = fmt.Sprintf("%s %q", tc.runner, "base")
				base = fmt.Sprintf("%s.%s.base", tc.kind, tc.runner)
			}

			source := fmt.Sprintf(`
%s %s {
  meta { name = "Inherited metadata" }
}
%s ref "derived" {
  base = %s
}
`, tc.kind, labels, tc.kind, base)

			parsedFile, err := ParseHclBytes(context.Background(), []byte(source), "meta.blackstork.hcl")
			require.NoError(t, err)

			var meta *definitions.MetaBlock
			if tc.kind == definitions.BlockKindSection {
				blockDef, ok := parsedFile.Blocks.GetSectionDefByName("derived")
				require.True(t, ok)
				block, diags := ParseSection(context.Background(), parsedFile.Blocks, blockDef, nil)
				require.False(t, diags.HasErrors(), diags.Error())
				meta = block.Meta
				require.Contains(t, blockDef.Block.Body.Attributes, definitions.AttrRefBase)
				baseDef, ok := parsedFile.Blocks.GetSectionDefByName("base")
				require.True(t, ok)
				require.True(t, hasMetaBlock(baseDef.Block.Body.Blocks))
			} else {
				key := definitions.Key{Kind: tc.kind, Runner: "ref", Name: "derived"}
				blockDef, ok := parsedFile.Blocks.GetExecBlockDefByKey(key)
				require.True(t, ok)

				switch tc.kind {
				case definitions.BlockKindData:
					parsed, diags := ParseDataBlock(context.Background(), parsedFile.Blocks, blockDef, nil)
					require.False(t, diags.HasErrors(), diags.Error())
					meta = parsed.Meta
				case definitions.BlockKindContent:
					parsed, diags := parseContentBlock(context.Background(), parsedFile.Blocks, blockDef, nil)
					require.False(t, diags.HasErrors(), diags.Error())
					meta = parsed.Meta
				case definitions.BlockKindFormat:
					parsed, diags := ParseFormatBlock(context.Background(), parsedFile.Blocks, blockDef, nil)
					require.False(t, diags.HasErrors(), diags.Error())
					meta = parsed.Meta
				case definitions.BlockKindPublish:
					parsed, diags := parsePublishBlock(context.Background(), parsedFile.Blocks, nil, blockDef, nil)
					require.False(t, diags.HasErrors(), diags.Error())
					meta = parsed.Meta
				}
				require.Contains(t, blockDef.Block.Body.Attributes, definitions.AttrRefBase)
				baseKey := definitions.Key{Kind: tc.kind, Runner: tc.runner, Name: "base"}
				baseDef, ok := parsedFile.Blocks.GetExecBlockDefByKey(baseKey)
				require.True(t, ok)
				require.True(t, hasMetaBlock(baseDef.Block.Body.Blocks))
			}

			require.NotNil(t, meta)
			require.Equal(t, "Inherited metadata", meta.Name)
		})
	}
}

func TestDocumentParsingDoesNotConsumeStandaloneSectionMeta(t *testing.T) {
	t.Parallel()

	parsedFile, err := ParseHclBytes(context.Background(), []byte(`
section "assessment" {
  meta { name = "Assessment metadata" }
}

document "report" {
  section ref {
    base = section.assessment
  }
}
`), "document-meta.blackstork.hcl")
	require.NoError(t, err)

	docDef, ok := parsedFile.Blocks.GetDocumentDefByName("report")
	require.True(t, ok)
	_, diags := ParseDocument(context.Background(), parsedFile.Blocks, docDef)
	require.False(t, diags.HasErrors(), diags.Error())

	sectionDef, ok := parsedFile.Blocks.GetSectionDefByName("assessment")
	require.True(t, ok)
	section, diags := ParseSection(context.Background(), parsedFile.Blocks, sectionDef, nil)
	require.False(t, diags.HasErrors(), diags.Error())
	require.NotNil(t, section.Meta)
	require.Equal(t, "Assessment metadata", section.Meta.Name)
}

func hasMetaBlock(blocks hclsyntax.Blocks) bool {
	for _, block := range blocks {
		if block.Type == definitions.BlockKindMeta {
			return true
		}
	}
	return false
}
