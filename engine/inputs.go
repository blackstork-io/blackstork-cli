// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package engine

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

func LoadInputs(ctx context.Context, doc *definitions.Document) (plugindata.Map, diagnostics.Diag) {
	log := appctx.Log(ctx)
	log = log.With("document", doc.Source.Name)

	cliInputs := appctx.Inputs(ctx)

	inputsData := make(plugindata.Map)

	if len(doc.Inputs) == 0 {
		log.DebugContext(ctx, "No inputs defined in the document")
	}

	for _, input := range doc.Inputs {

		_log := log.With("input_name", input.Name, "input_type", input.Type)

		// Read from CLI args

		var inputKV *appctx.InputKeyValue

		for _, kv := range cliInputs {
			if kv.Key == input.Name {
				inputKV = kv
				break
			}
		}
		if inputKV != nil {

			data, err := input.ParseValue(inputKV.Value)
			if err != nil {
				_log.ErrorContext(ctx, "Error while parsing input value", "err", err)
				return nil, diagnostics.Diag{
					{
						Severity: hcl.DiagError,
						Summary:  "Error while parsing input value from CLI argument",
						Detail:   err.Error(),
					},
				}
			}
			_log.InfoContext(ctx, "Using input value from CLI arguments")
			inputsData[input.Name] = data
			continue
		}

		// Read default value

		if !input.DefaultValue.IsNull() {
			_log.InfoContext(ctx, "Default value found for input")
			var err error
			val, err := input.ParseDefaultValue()
			if err != nil {
				_log.ErrorContext(ctx, "Error while parsing default value for input", "err", err)
				return nil, diagnostics.Diag{
					{
						Severity: hcl.DiagError,
						Summary:  "Error while parsing default value for input",
						Detail:   err.Error(),
						Subject:  &input.Block.Body.SrcRange,
					},
				}
			}
			inputsData[input.Name] = val
			continue
		}

		// Request inputs

		_log.InfoContext(ctx, "Requesting input value")

		// Requesting input value from stdin
		reader := bufio.NewReader(os.Stdin)

		var msg string
		if input.Label == nil {
			msg = fmt.Sprintf("Enter value for `%s`: ", input.Name)
		} else {
			msg = fmt.Sprintf("%s: ", *input.Label)
		}

		fmt.Print(msg)

		text, err := reader.ReadString('\n')
		if err != nil {
			return nil, diagnostics.Diag{
				{
					Severity: hcl.DiagError,
					Summary:  "Error reading input from stdin",
					Detail:   err.Error(),
				},
			}
		}

		valueStr := strings.TrimSpace(text)
		if valueStr == "" {
			_log.InfoContext(ctx, "No value provided for input")
			continue
		}

		val, err := input.ParseValue(valueStr)
		if err != nil {
			_log.ErrorContext(ctx, "Error while parsing input value", "err", err)
			return nil, diagnostics.Diag{
				{
					Severity: hcl.DiagError,
					Summary:  "Error while parsing input value",
					Detail:   err.Error(),
					Subject:  &input.Block.Body.SrcRange,
				},
			}
		}
		inputsData[input.Name] = val
	}
	return inputsData, nil
}
