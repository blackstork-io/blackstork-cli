package misp

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/misp/client"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

func makeMispEventsDataSource(loader ClientLoaderFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc:      "The `misp_events` data source fetches MISP events",
		DataFunc: fetchMispEventsData(loader),
		Config:   makeDataSourceConfig(),
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "value",
					Type:        cty.String,
					Constraints: constraint.Required,
				},
				{
					Name: "type",
					Type: cty.String,
				},
				{
					Name: "category",
					Type: cty.String,
				},
				{
					Name: "org",
					Type: cty.String,
				},
				{
					Name: "tags",
					Type: cty.List(cty.String),
				},
				{
					Name: "event_tags",
					Type: cty.List(cty.String),
				},
				{
					Name: "searchall",
					Type: cty.String,
				},
				{
					Name: "from",
					Type: cty.String,
				},
				{
					Name: "to",
					Type: cty.String,
				},
				{
					Name: "last",
					Type: cty.String,
				},
				{
					Name: "event_id",
					Type: cty.Number,
				},
				{
					Name: "with_attachments",
					Type: cty.Bool,
				},
				{
					Name: "sharing_groups",
					Type: cty.List(cty.String),
				},
				{
					Name: "only_metadata",
					Type: cty.Bool,
				},
				{
					Name: "uuid",
					Type: cty.String,
				},
				{
					Name: "include_sightings",
					Type: cty.Bool,
				},
				{
					Name: "threat_level_id",
					Type: cty.Number,
				},
				{
					Name:       "limit",
					Type:       cty.Number,
					DefaultVal: cty.NumberIntVal(10),
				},
			},
		},
	}
}

func fetchMispEventsData(loader ClientLoaderFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		cli := loader(params.Config)
		apiParams := makeRestSearchParams(params.Args)
		response, err := cli.RestSearchEvents(ctx, apiParams)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to fetch events",
				Detail:   err.Error(),
			}}
		}
		data, err := encodeResponse(response)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse response",
				Detail:   err.Error(),
			}}
		}
		return data, nil
	}
}

func makeRestSearchParams(args *dataspec.Block) (req client.RestSearchEventsRequest) {
	req.Value = args.GetAttrVal("value").AsString()
	typ := args.GetAttrVal("type")
	if !typ.IsNull() {
		req.Type = typ.AsString()
	}
	cat := args.GetAttrVal("category")
	if !cat.IsNull() {
		req.Category = cat.AsString()
	}
	org := args.GetAttrVal("org")
	if !org.IsNull() {
		req.Org = org.AsString()
	}
	tags := args.GetAttrVal("tags")
	if !tags.IsNull() {
		ctyTags := tags.AsValueSlice()
		for _, tag := range ctyTags {
			req.Tags = append(req.Tags, tag.AsString())
		}
	}
	eventTags := args.GetAttrVal("event_tags")
	if !eventTags.IsNull() {
		ctyEventTags := eventTags.AsValueSlice()
		for _, tag := range ctyEventTags {
			req.EventTags = append(req.EventTags, tag.AsString())
		}
	}
	searchAll := args.GetAttrVal("searchall")
	if !searchAll.IsNull() {
		req.SearchAll = searchAll.AsString()
	}
	from := args.GetAttrVal("from")
	if !from.IsNull() {
		fromStr := from.AsString()
		req.From = &fromStr
	}
	to := args.GetAttrVal("to")
	if !to.IsNull() {
		toStr := to.AsString()
		req.To = &toStr
	}
	last := args.GetAttrVal("last")
	if !last.IsNull() {
		lastStr := last.AsString()
		req.Last = &lastStr
	}
	eventID := args.GetAttrVal("event_id")
	if !eventID.IsNull() {
		req.EventID = eventID.AsString()
	}
	withAttachments := args.GetAttrVal("with_attachments")
	if !withAttachments.IsNull() {
		withAttachmentsBool := withAttachments.True()
		req.WithAttachments = &withAttachmentsBool
	}
	sharingGroups := args.GetAttrVal("sharing_groups")
	if !sharingGroups.IsNull() {
		ctySharingGroups := sharingGroups.AsValueSlice()
		for _, group := range ctySharingGroups {
			req.SharingGroups = append(req.SharingGroups, group.AsString())
		}
	}
	onlyMetadata := args.GetAttrVal("only_metadata")
	if !onlyMetadata.IsNull() {
		onlyMetadataBool := onlyMetadata.True()
		req.Metadata = &onlyMetadataBool
	}
	uuid := args.GetAttrVal("uuid")
	if !uuid.IsNull() {
		req.UUID = uuid.AsString()
	}

	includeSightings := args.GetAttrVal("include_sightings")
	if !includeSightings.IsNull() {
		includeSightingsBool := includeSightings.True()
		req.IncludeSightingdb = &includeSightingsBool
	}

	threatLevelID := args.GetAttrVal("threat_level_id")
	if !threatLevelID.IsNull() {
		req.ThreatLevelID = threatLevelID.AsString()
	}

	limit := args.GetAttrVal("limit")
	if !limit.IsNull() {
		limitInt, _ := limit.AsBigFloat().Int64()
		req.Limit = &limitInt
	}

	return
}
