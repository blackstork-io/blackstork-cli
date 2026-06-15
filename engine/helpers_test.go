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
	"context"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

type (
	optDocName      string
	optRequiredTags []string
)

func renderTest(t *testing.T, testName string, files, expectedResult []string, opts ...any) {
	t.Helper()
	sourceDir := fstest.MapFS{}
	for i, content := range files {
		sourceDir[fmt.Sprintf("file_%d.fabric", i)] = &fstest.MapFile{
			Data: []byte(content),
			Mode: 0o777,
		}
	}
	eng := New()
	ctx := appctx.New()

	target := "test-doc"
	var requiredTags []string
	diagAsserts := diagtest.Asserts{}

	for _, opt := range opts {
		switch v := opt.(type) {
		case diagtest.Asserts:
			diagAsserts = v
		case optDocName:
			target = string(v)
		case optRequiredTags:
			requiredTags = []string(v)
		default:
			t.Fatalf("unknown option type: %T", v)
		}
	}

	t.Run(testName, func(t *testing.T) {
		defer eng.Cleanup()

		var res string
		diags := eng.ParseDirFS(ctx, sourceDir)
		if !diags.HasErrors() {
			if !diags.Extend(eng.LoadPluginResolver(ctx, false)) && !diags.Extend(eng.LoadPluginRunner(ctx)) {
				_, content, _, diag := eng.RenderContent(ctx, target, requiredTags)
				if !diags.Extend(diag) {
					// FIXME:
					// res = mdprint.PrintString(content)
					t.Logf("ERROR: fix content check %s", content)
					res = "FIXME"
				}
			}
		}

		if len(expectedResult) == 0 {
			// so nil == []string{}
			assert.Empty(t, res)
		} else {
			ok := assert.EqualValues(
				t,
				strings.Join(expectedResult, "\n\n"),
				res,
			)
			if !ok {
				t.Logf("Got:\n\n%s", res)
			}
		}
		diagAsserts.AssertMatch(t, diags, eng.FileMap())
	})
}

func fetchDataTest(t *testing.T, testName string, files []string, target string, expectedResult plugindata.Map, diagAsserts diagtest.Asserts) {
	t.Helper()
	t.Run(testName, func(t *testing.T) {
		t.Helper()
		sourceDir := fstest.MapFS{}
		for i, content := range files {
			sourceDir[fmt.Sprintf("file_%d.fabric", i)] = &fstest.MapFile{
				Data: []byte(content),
				Mode: 0o777,
			}
		}
		eng := New()
		defer func() {
			eng.Cleanup()
		}()
		var res plugindata.Data
		ctx := context.Background()
		diags := eng.ParseDirFS(ctx, sourceDir)
		if !diags.HasErrors() {
			if !diags.Extend(eng.LoadPluginResolver(ctx, false)) && !diags.Extend(eng.LoadPluginRunner(ctx)) {
				var diag diagnostics.Diag
				res, diag = eng.FetchData(ctx, target)
				diags.Extend(diag)
			}
		}

		assert.Equal(t, expectedResult, res)
		diagAsserts.AssertMatch(t, diags, eng.FileMap())
	})
}

func lintTest(t *testing.T, fullLint bool, testName string, files []string, diagAsserts diagtest.Asserts) {
	t.Helper()
	t.Run(testName, func(t *testing.T) {
		t.Helper()
		sourceDir := fstest.MapFS{}
		for i, content := range files {
			sourceDir[fmt.Sprintf("file_%d.fabric", i)] = &fstest.MapFile{
				Data: []byte(content),
				Mode: 0o777,
			}
		}
		ctx := context.Background()

		eng := New()
		defer func() {
			eng.Cleanup()
		}()
		diag := []diagnostics.Diag{
			eng.ParseDirFS(ctx, sourceDir),
			eng.LoadPluginResolver(ctx, false),
			eng.LoadPluginRunner(ctx),
		}
		for _, d := range diag {
			if d.HasErrors() {
				t.Fatalf("Error: %v", d)
			}
		}
		diags := eng.Lint(ctx, fullLint)
		diagAsserts.AssertMatch(t, diags, eng.FileMap())
	})
}

func fullLintTest(t *testing.T, testName string, files []string, diagAsserts [][]diagtest.Assert) {
	t.Helper()
	lintTest(t, true, testName, files, diagAsserts)
}

func limitedLintTest(t *testing.T, testName string, files []string, diagAsserts [][]diagtest.Assert) {
	t.Helper()
	lintTest(t, false, testName, files, diagAsserts)
}
