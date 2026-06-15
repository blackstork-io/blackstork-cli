// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package plugins

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugins/atlassian"
	"github.com/blackstork-io/blackstork-cli/plugins/builtin"
	"github.com/blackstork-io/blackstork-cli/plugins/eclecticiq"
	"github.com/blackstork-io/blackstork-cli/plugins/elastic"
	"github.com/blackstork-io/blackstork-cli/plugins/github"
	"github.com/blackstork-io/blackstork-cli/plugins/graphql"
	"github.com/blackstork-io/blackstork-cli/plugins/hackerone"
	"github.com/blackstork-io/blackstork-cli/plugins/iris"
	"github.com/blackstork-io/blackstork-cli/plugins/microsoft"
	"github.com/blackstork-io/blackstork-cli/plugins/nistnvd"
	"github.com/blackstork-io/blackstork-cli/plugins/openai"
	"github.com/blackstork-io/blackstork-cli/plugins/opencti"
	"github.com/blackstork-io/blackstork-cli/plugins/postgresql"
	"github.com/blackstork-io/blackstork-cli/plugins/snyk"
	"github.com/blackstork-io/blackstork-cli/plugins/splunk"
	"github.com/blackstork-io/blackstork-cli/plugins/sqlite"
	"github.com/blackstork-io/blackstork-cli/plugins/stixview"
	"github.com/blackstork-io/blackstork-cli/plugins/terraform"
	"github.com/blackstork-io/blackstork-cli/plugins/virustotal"
)

// TestAllPluginSchemaValidity tests that all plugin schemas are valid
func TestAllPluginSchemaValidity(t *testing.T) {
	ver := "1.2.3"
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	plugins := []*plugin.Schema{
		builtin.Plugin(ver, logger, nil),
		elastic.Plugin(ver, nil),
		eclecticiq.Plugin(ver, nil),
		github.Plugin(ver, nil),
		graphql.Plugin(ver),
		openai.Plugin(ver, nil),
		opencti.Plugin(ver),
		postgresql.Plugin(ver),
		sqlite.Plugin(ver),
		terraform.Plugin(ver),
		hackerone.Plugin(ver, nil),
		virustotal.Plugin(ver, nil),
		stixview.Plugin(ver),
		splunk.Plugin(ver, nil),
		nistnvd.Plugin(ver, nil),
		snyk.Plugin(ver, nil),
		microsoft.Plugin(ver, nil, nil, nil),
		iris.Plugin(ver, nil),
		atlassian.Plugin(ver, nil),
	}
	for _, p := range plugins {
		p := p
		t.Run(p.Name, func(t *testing.T) {
			t.Parallel()
			assert.True(t, strings.HasPrefix(p.Name, "blackstork/"), "plugin name should be prefixed with 'blackstork/'")
			assert.Equal(t, ver, p.Version, "plugin version should match")
			assert.Greater(t, len(p.DataSources)+len(p.ContentProviders), 0, "plugin should have at least one data source or content provider")
			for name, ds := range p.DataSources {
				ds := ds
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					validateDataSource(t, ds)
				})
			}
			for name, cp := range p.ContentProviders {
				cp := cp
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					validateContentProvider(t, cp)
				})
			}
			for name, pub := range p.Publishers {
				pub := pub
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					validatePublisher(t, pub)
				})
			}
			assert.False(t, p.Validate().HasErrors(), "plugin should not have validation errors")
		})
	}
}

func validateDataSource(t testing.TB, ds *plugin.DataSource) {
	t.Helper()
	assert.NotNil(t, ds, "data source should not be nil")
	assert.NotEmpty(t, ds.DataFunc, "data source should have a data function")
	if ds.Config != nil {
		diagtest.AssertNoErrors(t, ds.Config.ValidateSpec(), nil, "data source config validation errors")
	}
	if ds.Args != nil {
		diagtest.AssertNoErrors(t, ds.Args.ValidateSpec(), nil, "data source args validation errors")
	}
}

func validateContentProvider(t testing.TB, cp *plugin.ContentProvider) {
	t.Helper()
	assert.NotNil(t, cp, "content provider should not be nil")
	assert.NotEmpty(t, cp.ContentFunc, "content provider should have a content function")
	if cp.Config != nil {
		diagtest.AssertNoErrors(t, cp.Config.ValidateSpec(), nil, "content provider config validation errors")
	}
	if cp.Args != nil {
		diagtest.AssertNoErrors(t, cp.Args.ValidateSpec(), nil, "content provider args validation errors")
	}
}

func validatePublisher(t testing.TB, pub *plugin.Publisher) {
	t.Helper()
	assert.NotNil(t, pub, "publisher should not be nil")
	assert.NotEmpty(t, pub.PublishFunc, "publisher should have a publish function")
	if pub.Config != nil {
		diagtest.AssertNoErrors(t, pub.Config.ValidateSpec(), nil, "publisher config validation errors")
	}
}
