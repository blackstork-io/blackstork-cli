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
	"testing"
	"testing/fstest"
)

func FuzzEngine(f *testing.F) {
	f.Add(
		[]byte(`
		document "test" {
			vars {
				a = 1
				b = query_jq(".vars.a + 1")
				xxx = "xxx"
			}
			content table {
				rows = query_jq("[10, 20, 10*(.vars.b + 1)]")

				columns = [
					{ header = "{{ .block.col_index }} Value", value = "{{ .block.row }}"},
					{ header = "{{ .block.col_index }} Index", value = "{{ .block.row_index }}"},
					{ header = "{{ .block.col_index }} ValueFromContext {{ .vars.xxx }}", value = "{{ .vars.xxx }}"},
					{ header = "{{ .block.col_index }} StaticValue", value = "foo"},
				]
			}
		}
		`),
		false,
		[]byte("document.test"),
	)

	f.Add(
		[]byte(`
		document "hello" {
			vars {
				a = 1
			}
			content text {
				value = "Hello: {{ .vars.a }}"
			}
		}
		`),
		false,
		[]byte("document.hello"),
	)

	f.Fuzz(func(t *testing.T, content []byte, publish bool, target []byte) {
		sourceDir := fstest.MapFS{}
		sourceDir["main.fabric"] = &fstest.MapFile{
			Data: content,
			Mode: 0o777,
		}

		eng := New()
		defer func() {
			eng.Cleanup()
		}()

		ctx := context.Background()
		diags := eng.ParseDirFS(ctx, sourceDir)
		if diags.HasErrors() {
			return
		}
		diags = eng.LoadPluginResolver(ctx, false)
		if diags.HasErrors() {
			return
		}
		diags = eng.LoadPluginRunner(ctx)
		if diags.HasErrors() {
			return
		}
		doc, renderedContent, dataCtx, diags := eng.RenderContent(ctx, string(target), []string{})
		if diags.HasErrors() {
			return
		}
		if publish {
			eng.PublishContent(ctx, string(target), doc, renderedContent, dataCtx)
		}
	})
}
