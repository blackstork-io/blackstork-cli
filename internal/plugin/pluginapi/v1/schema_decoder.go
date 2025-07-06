package pluginapiv1

import (
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
)

func decodeSchema(src *Schema) (*plugin.Schema, error) {
	if src == nil {
		return nil, nil
	}
	dataSources, err := decodeDataSourceSchemaMap(src.GetDataSources())
	if err != nil {
		return nil, err
	}

	contentProviders, err := decodeContentProviderSchemaMap(src.GetContentProviders())
	if err != nil {
		return nil, err
	}

	formatters, err := decodeFormatterSchemaMap(src.GetFormatters())
	if err != nil {
		return nil, err
	}

	publishers, err := decodePublisherSchemaMap(src.GetPublishers())
	if err != nil {
		return nil, err
	}
	return &plugin.Schema{
		Name:             src.GetName(),
		Version:          src.GetVersion(),
		DataSources:      dataSources,
		ContentProviders: contentProviders,
		Formatters:       formatters,
		Publishers:       publishers,
		Doc:              src.GetDoc(),
		Tags:             src.GetTags(),
	}, nil
}

func decodeDataSourceSchemaMap(src map[string]*DataSourceSchema) (plugin.DataSources, error) {
	dst := make(plugin.DataSources, len(src))
	var err error
	for k, v := range src {
		dst[k], err = decodeDataSourceSchema(v)
		if err != nil {
			return nil, err
		}
	}
	return dst, nil
}

func decodeDataSourceSchema(src *DataSourceSchema) (*plugin.DataSource, error) {
	if src == nil {
		return nil, nil
	}
	args, err := decodeBlockSpec(src.GetArgs())
	if err != nil {
		return nil, err
	}
	config, err := decodeBlockSpec(src.GetConfig())
	if err != nil {
		return nil, err
	}

	return &plugin.DataSource{
		Args:   dataspec.RootSpecFromBlock(args),
		Config: dataspec.RootSpecFromBlock(config),
		Doc:    src.GetDoc(),
		Tags:   src.GetTags(),
	}, nil
}

func decodeContentProviderSchemaMap(src map[string]*ContentProviderSchema) (plugin.ContentProviders, error) {
	dst := make(plugin.ContentProviders, len(src))
	var err error
	for k, v := range src {
		dst[k], err = decodeContentProviderSchema(v)
		if err != nil {
			return nil, err
		}
	}
	return dst, nil
}

func decodeContentProviderSchema(src *ContentProviderSchema) (*plugin.ContentProvider, error) {
	if src == nil {
		return nil, nil
	}
	args, err := decodeBlockSpec(src.GetArgs())
	if err != nil {
		return nil, err
	}
	config, err := decodeBlockSpec(src.GetConfig())
	if err != nil {
		return nil, err
	}
	return &plugin.ContentProvider{
		Args:   dataspec.RootSpecFromBlock(args),
		Config: dataspec.RootSpecFromBlock(config),
		Doc:    src.GetDoc(),
		Tags:   src.GetTags(),
	}, nil
}

func decodePublisherSchemaMap(src map[string]*PublisherSchema) (plugin.Publishers, error) {
	dst := make(plugin.Publishers, len(src))
	var err error
	for k, v := range src {
		dst[k], err = decodePublisherSchema(v)
		if err != nil {
			return nil, err
		}
	}
	return dst, nil
}

func decodePublisherSchema(src *PublisherSchema) (*plugin.Publisher, error) {
	if src == nil {
		return nil, nil
	}
	args, err := decodeBlockSpec(src.GetArgs())
	if err != nil {
		return nil, err
	}
	config, err := decodeBlockSpec(src.GetConfig())
	if err != nil {
		return nil, err
	}
	return &plugin.Publisher{
		Args:    dataspec.RootSpecFromBlock(args),
		Config:  dataspec.RootSpecFromBlock(config),
		Doc:     src.GetDoc(),
		Tags:    src.GetTags(),
		Formats: src.GetFormats(),
	}, nil
}

func decodeFormatterSchemaMap(src map[string]*FormatterSchema) (plugin.Formatters, error) {
	dst := make(plugin.Formatters, len(src))
	var err error
	for k, v := range src {
		dst[k], err = decodeFormatterSchema(v)
		if err != nil {
			return nil, err
		}
	}
	return dst, nil
}

func decodeFormatterSchema(src *FormatterSchema) (*plugin.Formatter, error) {
	if src == nil {
		return nil, nil
	}
	args, err := decodeBlockSpec(src.GetArgs())
	if err != nil {
		return nil, err
	}
	config, err := decodeBlockSpec(src.GetConfig())
	if err != nil {
		return nil, err
	}
	return &plugin.Formatter{
		Args:    dataspec.RootSpecFromBlock(args),
		Config:  dataspec.RootSpecFromBlock(config),
		Doc:     src.GetDoc(),
		Format:  src.GetFormat(),
		FileExt: src.GetFileExt(),
	}, nil
}
