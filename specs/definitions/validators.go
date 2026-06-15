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
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/blackstork-cli/pkg/utils"
)

func validateBlockName(block *hclsyntax.Block, idx int, required bool) *hcl.Diagnostic {
	if idx >= len(block.Labels) {
		if required {
			return &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing block name",
				Detail:   "Block name was not specified",
				Subject:  block.DefRange().Ptr(),
			}
		}
		return nil
	}

	if !hclsyntax.ValidIdentifier(block.Labels[idx]) {
		return &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid block name",
			Detail: fmt.Sprintf(
				"Block name '%s' is an invalid identifier",
				block.Labels[idx],
			),
			Subject: block.LabelRanges[idx].Ptr(),
			Context: block.DefRange().Ptr(),
		}
	}
	return nil
}

func validateExecBlockKind(block *hclsyntax.Block, kind string, kindRange hcl.Range) *hcl.Diagnostic {
	supportedKinds := []string{
		BlockKindContent, BlockKindData, BlockKindPublish, BlockKindFormat,
	}

	if slices.Contains(supportedKinds, kind) {
		return nil
	}

	kindsStr := strings.Join(supportedKinds, ", ")

	return &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Invalid block type",
		Detail: fmt.Sprintf(
			"Unknown block type `%s` does not match supported typese: %s",
			kind,
			kindsStr,
		),
		Subject: kindRange.Ptr(),
		Context: block.DefRange().Ptr(),
	}
}

func validateExecBlockKindLabel(block *hclsyntax.Block, idx int) *hcl.Diagnostic {
	if idx >= len(block.Labels) {
		return &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing block type",
			Detail:   "Block type is not specified",
			Subject:  block.DefRange().Ptr(),
		}
	}

	return validateExecBlockKind(block, block.Labels[idx], block.LabelRanges[idx])
}

func validateRunnerName(block *hclsyntax.Block, idx int) *hcl.Diagnostic {
	if idx >= len(block.Labels) {
		return &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing a block runner name",
			Detail:   "Either a data source, a content provider, a publisher or a formatter name must be provided",
			Subject:  block.DefRange().Ptr(),
		}
	}
	return nil
}

func validateLabelsLength(block *hclsyntax.Block, maxLabels int, labelUsage string) *hcl.Diagnostic {
	if len(block.Labels) > maxLabels {
		if labelUsage != "" {
			labelUsage = fmt.Sprintf("%s %s", block.Type, labelUsage)
		} else {
			labelUsage = block.Type
		}
		return &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Invalid %s block", block.Type),
			Detail:   fmt.Sprintf("Too many labels provided for the block. Usage: '%s'", labelUsage),
			Subject:  hcl.RangeBetween(block.LabelRanges[maxLabels], block.LabelRanges[len(block.LabelRanges)-1]).Ptr(),
			Context:  block.DefRange().Ptr(),
		}
	}
	return nil
}

func NewNestingDiag(what string, block *hclsyntax.Block, body *hclsyntax.Body, validChildren []string) *hcl.Diagnostic {
	return &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Invalid block type",
		Detail: fmt.Sprintf(
			"%s can't contain '%s' block, only %s",
			what,
			block.Type,
			utils.JoinSurround(", ", "'", validChildren...),
		),
		Subject: block.Range().Ptr(),
		Context: body.Range().Ptr(),
	}
}
