package opencti

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astvalidation"

	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

//go:embed opencti.graphql
var graphqlSchema string

var graphqlSchemaBase = `
schema {
	query: Query
}`

func Plugin(version string) *plugin.Schema {
	return &plugin.Schema{
		Name:    "blackstork/opencti",
		Version: version,
		DataSources: plugin.DataSources{
			"opencti": makeOpenCTIDataSource(),
		},
	}
}

type requestData struct {
	Query string `json:"query"`
}

func executeQuery(ctx context.Context, url, query, authToken string) (plugindata.Data, error) {
	data, err := json.Marshal(requestData{Query: query})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	// Set the appropriate headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return plugindata.UnmarshalJSON(raw)
}

func validateQuery(query string) error {
	schema, report := astparser.ParseGraphqlDocumentString(graphqlSchema + graphqlSchemaBase)
	if report.HasErrors() {
		return report
	}
	doc := ast.NewDocument()
	doc.Input.ResetInputString(query)
	astparser.NewParser().Parse(doc, &report)
	if report.HasErrors() {
		return report
	}
	validator := astvalidation.DefaultOperationValidator()
	validator.Validate(doc, &schema, &report)
	if report.HasErrors() {
		return report
	}
	return nil
}
