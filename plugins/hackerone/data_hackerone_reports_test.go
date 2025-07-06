package hackerone

import (
	"context"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/hackerone/client"
	client_mocks "github.com/blackstork-io/fabric/mocks/internalpkg/hackerone/client"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
	"github.com/blackstork-io/fabric/internal/plugin/plugintest"
)

type ReportsDataSourceTestSuite struct {
	suite.Suite
	schema    *plugin.DataSource
	ctx       context.Context
	cli       *client_mocks.Client
	storedUsr string
	storedTkn string
}

func TestReportsDataSourceTestSuite(t *testing.T) {
	suite.Run(t, new(ReportsDataSourceTestSuite))
}

func (s *ReportsDataSourceTestSuite) SetupSuite() {
	s.schema = makeHackerOneReportsDataSchema(func(user, token string) client.Client {
		s.storedUsr = user
		s.storedTkn = token
		return s.cli
	})
	s.ctx = context.Background()
}

func (s *ReportsDataSourceTestSuite) SetupTest() {
	s.cli = &client_mocks.Client{}
}

func (s *ReportsDataSourceTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *ReportsDataSourceTestSuite) TestSchema() {
	s.Require().NotNil(s.schema)
	s.NotNil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.DataFunc)
}

func (s *ReportsDataSourceTestSuite) TestPageNumber() {
	s.cli.On("GetAllReports", mock.Anything, &client.GetAllReportsReq{
		PageNumber:    client.Int(123),
		FilterProgram: []string{"test_program"},
	}).Return(&client.GetAllReportsRes{
		Data: []any{
			map[string]any{
				"id": "1",
			},
		},
	}, nil)
	res, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"api_username": cty.StringVal("test_user"),
			"api_token":    cty.StringVal("test_token"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"page_number": cty.NumberIntVal(123),
			"program":     cty.ListVal([]cty.Value{cty.StringVal("test_program")}),
		}),
	})
	s.Equal("test_user", s.storedUsr)
	s.Equal("test_token", s.storedTkn)
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.String("1"),
		},
	}, res)
}

func (s *ReportsDataSourceTestSuite) TestProgram() {
	s.cli.On("GetAllReports", mock.Anything, &client.GetAllReportsReq{
		PageNumber:    client.Int(1),
		FilterProgram: []string{"test_program"},
	}).Return(&client.GetAllReportsRes{
		Data: []any{
			map[string]any{
				"id": "1",
			},
		},
	}, nil)
	s.cli.On("GetAllReports", mock.Anything, &client.GetAllReportsReq{
		PageNumber:    client.Int(2),
		FilterProgram: []string{"test_program"},
	}).Return(&client.GetAllReportsRes{
		Data: []any{},
	}, nil)
	res, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"api_username": cty.StringVal("test_user"),
			"api_token":    cty.StringVal("test_token"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"program": cty.ListVal([]cty.Value{cty.StringVal("test_program")}),
		}),
	})
	s.Equal("test_user", s.storedUsr)
	s.Equal("test_token", s.storedTkn)
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.String("1"),
		},
	}, res)
}

func (s *ReportsDataSourceTestSuite) TestInboxIDs() {
	s.cli.On("GetAllReports", mock.Anything, &client.GetAllReportsReq{
		PageNumber:     client.Int(1),
		FilterInboxIDs: []int{1, 2, 3},
	}).Return(&client.GetAllReportsRes{
		Data: []any{
			map[string]any{
				"id": "1",
			},
		},
	}, nil)
	s.cli.On("GetAllReports", mock.Anything, &client.GetAllReportsReq{
		PageNumber:     client.Int(2),
		FilterInboxIDs: []int{1, 2, 3},
	}).Return(&client.GetAllReportsRes{
		Data: []any{},
	}, nil)
	res, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetHeaders("config").
			SetAttr("api_username", cty.StringVal("test_user")).
			SetAttr("api_token", cty.StringVal("test_token")).
			Decode(),

		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetHeaders("config").
			SetAttr("inbox_ids", cty.ListVal([]cty.Value{
				cty.NumberIntVal(1), cty.NumberIntVal(2), cty.NumberIntVal(3),
			})).
			Decode(),
	})
	s.Equal("test_user", s.storedUsr)
	s.Equal("test_token", s.storedTkn)
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.String("1"),
		},
	}, res)
}

func (s *ReportsDataSourceTestSuite) TestInvalid() {
	res, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"api_username": cty.StringVal("test_user"),
			"api_token":    cty.StringVal("test_token"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"size": cty.NumberIntVal(10),
		}),
	})
	s.Nil(res)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to parse arguments",
		Detail:   "at least one of program or inbox_ids must be provided",
	}}, diags)
}

func (s *ReportsDataSourceTestSuite) TestInvalidConfig() {
	res, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{}),
		Args:   dataspec.NewBlock([]string{"args"}, map[string]cty.Value{}),
	})
	s.Nil(res)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to create client",
		Detail:   "api_username is required in configuration",
	}}, diags)
}
