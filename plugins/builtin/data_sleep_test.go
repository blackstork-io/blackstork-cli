package builtin

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
	"github.com/blackstork-io/fabric/internal/plugin/plugintest"
)

type SleepDataTestSuite struct {
	suite.Suite
	schema *plugin.DataSource
}

func TestSleepDataSuite(t *testing.T) {
	suite.Run(t, &SleepDataTestSuite{})
}

func (s *SleepDataTestSuite) SetupSuite() {
	s.schema = makeSleepDataSource(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func (s *SleepDataTestSuite) TestSchema() {
	s.Nil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.DataFunc)
}

func (s *SleepDataTestSuite) TestMissingDuration() {
	plugintest.NewTestDecoder(s.T(), s.schema.Args).Decode()
}

func (s *SleepDataTestSuite) TestCustom() {
	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("duration", cty.StringVal("123ms")).
			Decode(),
	})
	s.Empty(diags)

	s.Require().IsType(plugindata.Map{}, result)

	resultMap := result.(plugindata.Map)
	s.Require().Equal(plugindata.String("123ms"), resultMap["took"])
	s.NotEmpty(resultMap["start_time"])
	s.NotEmpty(resultMap["end_time"])
}

func (s *SleepDataTestSuite) TestDefault() {
	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).Decode(),
	})
	s.Require().Empty(diags)

	s.Require().IsType(plugindata.Map{}, result)

	resultMap := result.(plugindata.Map)
	s.Require().Equal(plugindata.String("1s"), resultMap["took"])
	s.NotEmpty(resultMap["start_time"])
	s.NotEmpty(resultMap["end_time"])
}

func (s *SleepDataTestSuite) TestCallInvalidDuration() {
	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("duration", cty.StringVal("invalid")).
			Decode(),
	})
	s.Require().Nil(result)
	s.Require().Len(diags, 1)
	s.Equal(hcl.DiagError, diags[0].Severity)
	s.Equal("Invalid duration", diags[0].Summary)
}
