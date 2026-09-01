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
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/xeipuuv/gojsonschema"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

var supportedInputTypes = []string{
	"bool", "datetime", "number", "json", "string", "secret",
}

type InputBlock struct {
	Name string // set from a label
	Type string `hcl:"type"`

	Label        *string   `hcl:"label,optional"`
	Description  *string   `hcl:"description,optional"`
	DefaultValue cty.Value `hcl:"default_value,optional"`
	Schema       cty.Value `hcl:"schema,optional"`

	Block *hclsyntax.Block
}

func (b *InputBlock) IsTypeValid() bool {
	return slices.Contains(supportedInputTypes, b.Type)
}

func (b *InputBlock) getJSONSchemaLoader() (gojsonschema.JSONLoader, error) {
	schemaJSON, err := ctyjson.Marshal(b.Schema, b.Schema.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %s", err)
	}

	loader := gojsonschema.NewBytesLoader(schemaJSON)
	return loader, nil
}

func (b *InputBlock) validateJSON(value any) error {
	if b.Schema.IsNull() {
		return nil
	}

	schemaLoader, err := b.getJSONSchemaLoader()
	if err != nil {
		return err
	}

	result, err := gojsonschema.Validate(schemaLoader, gojsonschema.NewGoLoader(value))
	if err != nil {
		return err
	}
	if result.Valid() {
		return nil
	}

	errs := strings.Join(
		utils.FnMap(result.Errors(), func(e gojsonschema.ResultError) string {
			return e.String()
		}),
		"; ",
	)
	return fmt.Errorf("validation failed: %s", errs)
}

// ParseDefaultValue converts and validates the input's HCL default value.
func (b *InputBlock) ParseDefaultValue() (plugindata.Data, error) {
	if b.DefaultValue.IsNull() || !b.DefaultValue.IsKnown() {
		return nil, nil
	}

	if b.Type != "json" {
		return plugindata.CtyToPluginDataWithType(b.DefaultValue, b.Type)
	}

	value, err := plugindata.CtyToPluginData(b.DefaultValue)
	if err != nil {
		return nil, err
	}
	if err := b.validateJSON(value.Any()); err != nil {
		return nil, err
	}
	return value, nil
}

func (b *InputBlock) ParseValue(value string) (plugindata.Data, error) {
	// Trim whitespace to avoid common parsing errors
	value = strings.TrimSpace(value)

	switch b.Type {

	case "string", "secret":
		return plugindata.String(value), nil

	case "bool":
		// Standard lib: accepts 1, t, T, TRUE, true, 0, f, F, FALSE, false
		boolV, err := strconv.ParseBool(value)
		if err != nil {
			return nil, err
		}
		return plugindata.Bool(boolV), nil

	case "datetime":
		// RFC3339 is the standard implementation for ISO8601
		val, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, err
		}
		return plugindata.Time(val), nil

	case "number":
		// Standard 64-bit precision float
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		return plugindata.Number(val), nil

	case "json":

		var v any
		if err := json.Unmarshal([]byte(value), &v); err != nil {
			return nil, err
		}

		if err := b.validateJSON(v); err != nil {
			return nil, err
		}
		return plugindata.ParseAny(v)
	default:
		return nil, fmt.Errorf("unsupported type: %s", b.Type)
	}
}
