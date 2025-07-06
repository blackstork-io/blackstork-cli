package elastic

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	es "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
	"github.com/blackstork-io/fabric/plugins/elastic/kbclient"
)

const (
	defaultUsername       = "elastic"
	defaultScrollStepSize = 1000
)

type KibanaClientLoaderFn func(url string, apiKey *string) kbclient.Client

var DefaultKibanaClientLoader KibanaClientLoaderFn = kbclient.New

func Plugin(version string, loader KibanaClientLoaderFn) *plugin.Schema {
	if loader == nil {
		loader = DefaultKibanaClientLoader
	}
	return &plugin.Schema{
		Name:    "blackstork/elastic",
		Version: version,
		DataSources: plugin.DataSources{
			"elasticsearch":          makeElasticSearchDataSource(),
			"elastic_security_cases": makeElasticSecurityCasesDataSource(loader),
		},
	}
}

func makeSearchClient(pcfg *dataspec.Block) (*es.Client, error) {
	cfg := &es.Config{
		Username: defaultUsername,
	}
	if pcfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	if baseURL := pcfg.GetAttrVal("base_url"); !baseURL.IsNull() && baseURL.AsString() != "" {
		cfg.Addresses = []string{baseURL.AsString()}
	}
	if cloudID := pcfg.GetAttrVal("cloud_id"); !cloudID.IsNull() {
		cfg.CloudID = cloudID.AsString()
	}
	if len(cfg.Addresses) == 0 && cfg.CloudID == "" {
		return nil, fmt.Errorf("either one of base_url or cloud_id is required")
	}
	if apiKeyStr := pcfg.GetAttrVal("api_key_str"); !apiKeyStr.IsNull() {
		cfg.APIKey = apiKeyStr.AsString()
	}
	if apiKey := pcfg.GetAttrVal("api_key"); !apiKey.IsNull() {
		list := apiKey.AsValueSlice()
		if len(list) != 2 {
			return nil, fmt.Errorf("api_key must be a list of 2 strings")
		}
		cfg.APIKey = base64.StdEncoding.EncodeToString([]byte(
			fmt.Sprintf("%s:%s", list[0].AsString(), list[1].AsString())),
		)
	}
	if basicAuthUsername := pcfg.GetAttrVal("basic_auth_username"); !basicAuthUsername.IsNull() {
		cfg.Username = basicAuthUsername.AsString()
	}
	if basicAuthPassword := pcfg.GetAttrVal("basic_auth_password"); !basicAuthPassword.IsNull() {
		cfg.Password = basicAuthPassword.AsString()
	}
	if bearerAuth := pcfg.GetAttrVal("bearer_auth"); !bearerAuth.IsNull() {
		cfg.ServiceToken = bearerAuth.AsString()
	}
	if caCerts := pcfg.GetAttrVal("ca_certs"); !caCerts.IsNull() {
		cfg.CACert = []byte(caCerts.AsString())
	}
	return es.NewClient(*cfg)
}

func getByID(fn esapi.Get, args *dataspec.Block) (plugindata.Data, error) {
	index := args.GetAttrVal("index")
	id := args.GetAttrVal("id")
	if id.IsNull() {
		return nil, fmt.Errorf("id is required when id is specified")
	}
	opts := []func(*esapi.GetRequest){}
	if fields := args.GetAttrVal("fields"); !fields.IsNull() {
		fieldSlice := fields.AsValueSlice()
		fieldStrings := make([]string, len(fieldSlice))
		for i, v := range fieldSlice {
			fieldStrings[i] = v.AsString()
		}
		opts = append(opts, fn.WithSource(fieldStrings...))
	}
	res, err := fn(index.AsString(), id.AsString(), opts...)
	if err != nil {
		return nil, err
	}
	if res.IsError() {
		return nil, fmt.Errorf("failed to get document: %s", res.String())
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read search result: %s", err)
	}
	data, err := plugindata.UnmarshalJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal search result: %s", err)
	}
	if onlyHits := args.GetAttrVal("only_hits"); onlyHits.IsNull() || onlyHits.True() {
		return extractHits(data)
	}
	return data, nil
}

func unpackSearchOptions(fn esapi.Search, args *dataspec.Block) ([]func(*esapi.SearchRequest), error) {
	index := args.GetAttrVal("index")
	opts := []func(*esapi.SearchRequest){
		fn.WithIndex(index.AsString()),
	}
	if queryString := args.GetAttrVal("query_string"); !queryString.IsNull() {
		opts = append(opts, fn.WithQuery(queryString.AsString()))
	}
	body := map[string]any{}
	if query := args.GetAttrVal("query"); !query.IsNull() {
		raw, err := ctyjson.Marshal(query, query.Type())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal query: %s", err)
		}
		body["query"] = json.RawMessage(raw)

	}
	if aggs := args.GetAttrVal("aggs"); !aggs.IsNull() {
		raw, err := ctyjson.Marshal(aggs, aggs.Type())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal aggs: %s", err)
		}
		body["aggs"] = json.RawMessage(raw)
	}
	if len(body) > 0 {
		rawBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal query: %s", err)
		}
		opts = append(opts, fn.WithBody(bytes.NewReader(rawBody)))
	}
	if fields := args.GetAttrVal("fields"); !fields.IsNull() {
		fieldSlice := fields.AsValueSlice()
		fieldStrings := make([]string, len(fieldSlice))
		for i, v := range fieldSlice {
			fieldStrings[i] = v.AsString()
		}
		opts = append(opts, fn.WithSource(fieldStrings...))
	}
	return opts, nil
}

func unpackResponse(response *esapi.Response) (plugindata.Map, error) {
	// Read the response
	raw, err := io.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read the search response: %s", err)
	}
	data, err := plugindata.UnmarshalJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal the search response: %s", err)
	}
	mapData, ok := data.(plugindata.Map)
	if !ok {
		return nil, fmt.Errorf("unexpected search result type: %T", data)
	}
	return mapData, nil
}

func searchWithScroll(client *es.Client, args *dataspec.Block, size int) (plugindata.Data, error) {
	return searchWithScrollConfigurable(client, args, size, defaultScrollStepSize)
}

func searchWithScrollConfigurable(
	client *es.Client,
	args *dataspec.Block,
	size, stepSize int,
) (plugindata.Data, error) {
	// First scroll request to obtain `_scroll_id`
	scrollTTL := time.Minute * 2 // 2mins
	// just in case the size is smaller than the step size
	reqSize := min(size, stepSize)

	requestsCounter := 0

	opts, err := unpackSearchOptions(client.Search, args)
	if err != nil {
		return nil, err
	}
	opts = append(opts, client.Search.WithScroll(scrollTTL))
	opts = append(opts, client.Search.WithSize(reqSize))

	slog.Debug(
		"Sending an initial scroll search request",
		"size", reqSize,
		"scroll_ttl", scrollTTL,
	)
	requestsCounter += 1
	res, err := client.Search(opts...)
	if err != nil {
		return nil, err
	} else if res.IsError() {
		return nil, fmt.Errorf("failed to perform a search: %s", res.String())
	}

	firstData, err := unpackResponse(res)
	if err != nil {
		return nil, err
	}
	scrollIDRaw, ok := firstData["_scroll_id"]
	if !ok {
		return nil, fmt.Errorf("error while getting scroll id value: %s", firstData)
	}
	scrollID := scrollIDRaw.(plugindata.String)

	hitsEnvelope, ok := firstData["hits"].(plugindata.Map)
	if !ok {
		return nil, fmt.Errorf("unexpected hits envelope value type: %T", firstData)
	}
	hits, ok := hitsEnvelope["hits"].(plugindata.List)
	if !ok {
		return nil, fmt.Errorf("unexpected hits value type: %T", firstData)
	}
	allHits := make(plugindata.List, 0, len(hits))
	allHits = append(allHits, hits...)

	for {
		if len(allHits) >= size {
			break
		}

		// just in case the leftover size is smaller than the step size
		reqSize := min((size - len(allHits)), stepSize)
		if reqSize <= 0 {
			break
		}

		slog.Debug(
			"Sending a scroll search request",
			"scroll_id", scrollID,
			"scroll_ttl", scrollTTL,
			"size", reqSize,
			"hits_count", len(allHits),
		)
		requestsCounter += 1
		res, err = client.Scroll(
			client.Scroll.WithScrollID(string(scrollID)),
			client.Scroll.WithScroll(scrollTTL),
		)

		if err != nil {
			return nil, err
		} else if res.IsError() {
			return nil, fmt.Errorf("failed to perform a scroll search: %s", res.String())
		}

		scrollData, err := unpackResponse(res)
		if err != nil {
			return nil, err
		}
		scrollIDRaw, ok = scrollData["_scroll_id"]
		if !ok {
			return nil, fmt.Errorf("error while getting scroll id value: %s", scrollData)
		}
		scrollID = scrollIDRaw.(plugindata.String)

		hitsEnvelope, ok := scrollData["hits"].(plugindata.Map)
		if !ok {
			return nil, fmt.Errorf("unexpected hits envelope value type: %T", firstData)
		}
		hits, ok := hitsEnvelope["hits"].(plugindata.List)
		if !ok {
			return nil, fmt.Errorf("unexpected hits value type: %T", scrollData)
		}

		if len(hits) == 0 {
			break
		}
		allHits = append(allHits, hits...)
	}

	slog.Debug(
		"Scroll search finished",
		"hits_count", len(allHits),
		"requests_count", requestsCounter,
	)
	if onlyHits := args.GetAttrVal("only_hits"); onlyHits.IsNull() || onlyHits.True() {
		return allHits, nil
	}
	// Update and return the first returned data to keep the aggregations
	hitsEnvelope["hits"] = allHits
	return firstData, nil
}

func search(fn esapi.Search, args *dataspec.Block, size int) (plugindata.Data, error) {
	opts, err := unpackSearchOptions(fn, args)
	if err != nil {
		return nil, err
	}
	opts = append(opts, fn.WithSize(size))

	res, err := fn(opts...)
	if err != nil {
		return nil, err
	} else if res.IsError() {
		return nil, fmt.Errorf("failed to search: %s", res.String())
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read search result: %s", err)
	}
	data, err := plugindata.UnmarshalJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal search result: %s", err)
	}
	if onlyHits := args.GetAttrVal("only_hits"); onlyHits.IsNull() || onlyHits.True() {
		return extractHits(data)
	}
	return data, nil
}

func extractHits(data plugindata.Data) (plugindata.List, error) {
	// Unpack the result as a map
	m, ok := data.(plugindata.Map)
	if !ok {
		return nil, fmt.Errorf("unexpected search result type: %T", data)
	}
	data, ok = m["hits"]
	if !ok {
		return nil, fmt.Errorf("unexpected search result type: %T", data)
	}
	// Unpack "hits" value as a map
	m, ok = data.(plugindata.Map)
	if !ok {
		return nil, fmt.Errorf("unexpected search result type: %T", data)
	}
	return m["hits"].(plugindata.List), nil
}
