// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package nistnvd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	client_mocks "github.com/blackstork-io/blackstork-cli/mocks/plugins/nistnvd/client"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/nistnvd/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

type CVESDataSourceTestSuite struct {
	suite.Suite
	schema       *plugin.DataSource
	ctx          context.Context
	cli          *client_mocks.Client
	storedApiKey *string
}

func TestCVESDataSourceTestSuite(t *testing.T) {
	suite.Run(t, new(CVESDataSourceTestSuite))
}

func (s *CVESDataSourceTestSuite) SetupSuite() {
	s.schema = makeNistNvdCvesDataSource(func(apiKey *string) client.Client {
		s.storedApiKey = apiKey
		return s.cli
	})
	s.ctx = context.Background()
}

func (s *CVESDataSourceTestSuite) SetupTest() {
	s.cli = &client_mocks.Client{}
}

func (s *CVESDataSourceTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *CVESDataSourceTestSuite) TestSchema() {
	s.Require().NotNil(s.schema)
	s.NotNil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.DataFunc)
}

func (s *CVESDataSourceTestSuite) TestLimit() {
	s.cli.On("ListCVES", mock.Anything, &client.ListCVESReq{
		ResultsPerPage: 123,
		StartIndex:     0,
	}).Return(&client.ListCVESRes{
		ResultsPerPage: 123,
		StartIndex:     0,
		TotalResults:   1,
		Vulnerabilities: []any{
			map[string]any{
				"id": "1",
			},
		},
	}, nil)
	res, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"api_key": cty.StringVal("test_key"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"limit": cty.NumberIntVal(123),
		}),
	})
	s.Equal("test_key", *s.storedApiKey)
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.String("1"),
		},
	}, res)
}

func (s *CVESDataSourceTestSuite) TestFull() {
	s.cli.On("ListCVES", mock.Anything, &client.ListCVESReq{
		ResultsPerPage:     1,
		StartIndex:         0,
		LastModStartDate:   client.String("2021-01-01T00:00:00Z"),
		LastModEndDate:     client.String("2021-01-02T00:00:00Z"),
		PubStartDate:       client.String("2021-01-03T00:00:00Z"),
		PubEndDate:         client.String("2021-01-04T00:00:00Z"),
		CPEName:            client.String("cpe:2.3:o:microsoft:windows_10:1607"),
		CVEID:              client.String("cve-2021-1234"),
		CVSSV3Metrics:      client.String("cvssv3"),
		CVSSV3Severity:     client.String("high"),
		VirtualMatchString: client.String("virtual"),
		CWEID:              client.String("cwe-123"),
		HasCertAlerts:      client.Bool(true),
		HasCertNotes:       client.Bool(true),
		HasKev:             client.Bool(true),
		IsVulnerable:       client.Bool(true),
		NoRejected:         client.Bool(true),
		KeywordSearch:      client.String("keyword"),
		KeywordExactMatch:  client.Bool(true),
		SourceIdentifier:   client.String("source"),
	}).Return(&client.ListCVESRes{
		ResultsPerPage: 1,
		StartIndex:     0,
		TotalResults:   2,
		Vulnerabilities: []any{
			map[string]any{
				"id": "1",
			},
		},
	}, nil)
	s.cli.On("ListCVES", mock.Anything, &client.ListCVESReq{
		ResultsPerPage:     1,
		StartIndex:         1,
		LastModStartDate:   client.String("2021-01-01T00:00:00Z"),
		LastModEndDate:     client.String("2021-01-02T00:00:00Z"),
		PubStartDate:       client.String("2021-01-03T00:00:00Z"),
		PubEndDate:         client.String("2021-01-04T00:00:00Z"),
		CPEName:            client.String("cpe:2.3:o:microsoft:windows_10:1607"),
		CVEID:              client.String("cve-2021-1234"),
		VirtualMatchString: client.String("virtual"),
		CVSSV3Metrics:      client.String("cvssv3"),
		CVSSV3Severity:     client.String("high"),
		CWEID:              client.String("cwe-123"),
		HasCertAlerts:      client.Bool(true),
		HasCertNotes:       client.Bool(true),
		HasKev:             client.Bool(true),
		IsVulnerable:       client.Bool(true),
		NoRejected:         client.Bool(true),
		KeywordSearch:      client.String("keyword"),
		KeywordExactMatch:  client.Bool(true),
		SourceIdentifier:   client.String("source"),
	}).Return(&client.ListCVESRes{
		ResultsPerPage: 1,
		StartIndex:     1,
		TotalResults:   2,
		Vulnerabilities: []any{
			map[string]any{
				"id": "2",
			},
		},
	}, nil)
	res, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"api_key": cty.StringVal("test_key"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"last_mod_start_date":  cty.StringVal("2021-01-01T00:00:00Z"),
			"last_mod_end_date":    cty.StringVal("2021-01-02T00:00:00Z"),
			"pub_start_date":       cty.StringVal("2021-01-03T00:00:00Z"),
			"pub_end_date":         cty.StringVal("2021-01-04T00:00:00Z"),
			"cpe_name":             cty.StringVal("cpe:2.3:o:microsoft:windows_10:1607"),
			"cve_id":               cty.StringVal("cve-2021-1234"),
			"cvss_v3_metrics":      cty.StringVal("cvssv3"),
			"cvss_v3_severity":     cty.StringVal("high"),
			"cwe_id":               cty.StringVal("cwe-123"),
			"keyword_search":       cty.StringVal("keyword"),
			"virtual_match_string": cty.StringVal("virtual"),
			"source_identifier":    cty.StringVal("source"),
			"has_cert_alerts":      cty.BoolVal(true),
			"has_kev":              cty.BoolVal(true),
			"has_cert_notes":       cty.BoolVal(true),
			"is_vulnerable":        cty.BoolVal(true),
			"keyword_exact_match":  cty.BoolVal(true),
			"no_rejected":          cty.BoolVal(true),
			"limit":                cty.NumberIntVal(1),
		}),
	})
	s.Equal("test_key", *s.storedApiKey)
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.String("1"),
		},
		plugindata.Map{
			"id": plugindata.String("2"),
		},
	}, res)
}
