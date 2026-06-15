// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package main

import (
	"cmp"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/spf13/pflag"

	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugins/atlassian"
	"github.com/blackstork-io/blackstork-cli/plugins/builtin"
	"github.com/blackstork-io/blackstork-cli/plugins/crowdstrike"
	"github.com/blackstork-io/blackstork-cli/plugins/eclecticiq"
	"github.com/blackstork-io/blackstork-cli/plugins/elastic"
	"github.com/blackstork-io/blackstork-cli/plugins/github"
	"github.com/blackstork-io/blackstork-cli/plugins/graphql"
	"github.com/blackstork-io/blackstork-cli/plugins/hackerone"
	"github.com/blackstork-io/blackstork-cli/plugins/iris"
	"github.com/blackstork-io/blackstork-cli/plugins/microsoft"
	"github.com/blackstork-io/blackstork-cli/plugins/misp"
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
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

var (
	version   string
	outputDir string
)

//go:embed content-provider.md.gotempl
var contentProviderTemplValue string

var base *template.Template

// var contentProviderTempl *template.Template

//go:embed data-source.md.gotempl
var dataSourceTemplValue string

// var dataSourceTempl *template.Template

//go:embed plugin.md.gotempl
var pluginTemplValue string

//go:embed publisher.md.gotempl
var publisherTemplValue string

//go:embed formatter.md.gotempl
var formatterTemplValue string

type PluginResourceMeta struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	ConfigParams []string `json:"config_params,omitempty"`
	Arguments    []string `json:"arguments,omitempty"`
}

type PluginDetails struct {
	Name      string               `json:"name"`
	Version   string               `json:"version"`
	Shortname string               `json:"shortname"`
	Resources []PluginResourceMeta `json:"resources"`
}

func generateDataSourceDocs(log *slog.Logger, p *plugin.Schema, outputDir string) {
	log.Info("Found data sources inside the plugin", "count", len(p.DataSources))

	dataSourcesDir := filepath.Join(outputDir, "data-sources")

	// Create a directory for plugin's data sources if it doesn't exist
	err := os.MkdirAll(dataSourcesDir, 0o750)
	if err != nil {
		log.Error("Can't create a directory", "path", dataSourcesDir)
		panic(err)
	}

	for name, dataSource := range p.DataSources {
		log.Info("Found a data source", "name", name)
		docFilename := fmt.Sprintf("%s.md", name)
		docPath := filepath.Join(dataSourcesDir, docFilename)
		err := renderDataSourceDoc(p, name, dataSource, docPath)
		if err != nil {
			log.Error("Error while rendering a data source doc", "plugin", p.Name, "dataSource", name)
			panic(err)
		}
	}
}

func generateContentProviderDocs(log *slog.Logger, p *plugin.Schema, outputDir string) {
	log.Info("Found content providers inside the plugin", "count", len(p.ContentProviders))

	contentProvidersDir := filepath.Join(outputDir, "content-providers")

	// Create a directory for plugin's content providers if it doesn't exist
	err := os.MkdirAll(contentProvidersDir, 0o750)
	if err != nil {
		log.Error("Can't create a directory", "path", contentProvidersDir)
		panic(err)
	}

	for name, contentProvider := range p.ContentProviders {
		log.Info("Found a content provider", "name", name)

		docFilename := fmt.Sprintf("%s.md", name)
		docPath := filepath.Join(contentProvidersDir, docFilename)
		err := renderContentProviderDoc(p, name, contentProvider, docPath)
		if err != nil {
			log.Error("Error while rendering a content provider doc", "plugin", p.Name, "contentProvider", name)
			panic(err)
		}
	}
}

func generatePublisherDocs(log *slog.Logger, p *plugin.Schema, outputDir string) {
	log.Info("Found publishers inside the plugin", "count", len(p.Publishers))

	publishersDir := filepath.Join(outputDir, "publishers")

	// Create a directory for plugin's publishers if it doesn't exist
	err := os.MkdirAll(publishersDir, 0o750)
	if err != nil {
		log.Error("Can't create a directory", "path", publishersDir)
		panic(err)
	}

	for name, publisher := range p.Publishers {
		log.Info("Found a publisher", "name", name)

		docFilename := fmt.Sprintf("%s.md", name)
		docPath := filepath.Join(publishersDir, docFilename)
		err := renderPublisherDoc(p, name, publisher, docPath)
		if err != nil {
			log.Error("Error while rendering a publisher doc", "plugin", p.Name, "publisher", name)
			panic(err)
		}
	}
}

func generateFormatterDocs(log *slog.Logger, p *plugin.Schema, outputDir string) {
	log.Info("Found formatter inside the plugin", "count", len(p.Formatters))

	formattersDir := filepath.Join(outputDir, "formatters")

	// Create a directory for plugin's formatters if it doesn't exist
	err := os.MkdirAll(formattersDir, 0o750)
	if err != nil {
		log.Error("Can't create a directory", "path", formattersDir)
		panic(err)
	}

	for name, formatter := range p.Formatters {
		log.Info("Found a formatter", "name", name)

		docFilename := fmt.Sprintf("%s.md", name)
		docPath := filepath.Join(formattersDir, docFilename)
		err := renderFormatterDoc(p, name, formatter, docPath)
		if err != nil {
			log.Error("Error while rendering a formatter doc", "plugin", p.Name, "formatter", name)
			panic(err)
		}
	}
}

func dumpAttrNames(spec *dataspec.RootSpec) []string {
	if spec == nil {
		return nil
	}
	return utils.FnMap(
		spec.Attrs,
		func(a *dataspec.AttrSpec) string {
			return a.Name
		},
	)
}

func marshalDataSource(name string, ds *plugin.DataSource) PluginResourceMeta {
	configParams := dumpAttrNames(ds.Config)
	slices.Sort(configParams)

	arguments := dumpAttrNames(ds.Args)
	slices.Sort(arguments)

	return PluginResourceMeta{
		Name:         name,
		Type:         "data-source",
		ConfigParams: configParams,
		Arguments:    arguments,
	}
}

func marshalContentProvider(name string, p *plugin.ContentProvider) PluginResourceMeta {
	configParams := dumpAttrNames(p.Config)
	slices.Sort(configParams)

	arguments := dumpAttrNames(p.Args)
	slices.Sort(arguments)

	return PluginResourceMeta{
		Name:         name,
		Type:         "content-provider",
		ConfigParams: configParams,
		Arguments:    arguments,
	}
}

func marshalFormatter(name string, p *plugin.Formatter) PluginResourceMeta {
	configParams := dumpAttrNames(p.Config)
	slices.Sort(configParams)

	arguments := dumpAttrNames(p.Args)
	slices.Sort(arguments)

	return PluginResourceMeta{
		Name:         name,
		Type:         "formatter",
		ConfigParams: configParams,
		Arguments:    arguments,
	}
}

func marshalPublisher(name string, p *plugin.Publisher) PluginResourceMeta {
	configParams := dumpAttrNames(p.Config)
	slices.Sort(configParams)

	arguments := dumpAttrNames(p.Args)
	slices.Sort(arguments)

	return PluginResourceMeta{
		Name:         name,
		Type:         "publisher",
		ConfigParams: configParams,
		Arguments:    arguments,
	}
}

func generateMetadataFile(plugins []*plugin.Schema, outputDir string) {
	pluginDetails := make([]PluginDetails, len(plugins))

	for i, p := range plugins {

		var resources []PluginResourceMeta

		for name, dataSource := range p.DataSources {
			resources = append(resources, marshalDataSource(name, dataSource))
		}
		for name, contentProvider := range p.ContentProviders {
			resources = append(resources, marshalContentProvider(name, contentProvider))
		}
		for name, formatter := range p.Formatters {
			resources = append(resources, marshalFormatter(name, formatter))
		}
		for name, publisher := range p.Publishers {
			resources = append(resources, marshalPublisher(name, publisher))
		}

		sort.Slice(resources, func(i, j int) bool {
			a := resources[i]
			b := resources[j]
			return cmp.Or(
				cmp.Compare(a.Name, b.Name),
				cmp.Compare(a.Type, b.Type),
			) < 0
		})

		pluginDetails[i] = PluginDetails{
			Name:      p.Name,
			Version:   p.Version,
			Shortname: p.Shortname(),
			Resources: resources,
		}
	}

	sort.Slice(pluginDetails, func(i, j int) bool {
		return pluginDetails[i].Name < pluginDetails[j].Name
	})

	jsonData, err := json.MarshalIndent(pluginDetails, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal the plugin details into JSON")
		return
	}

	pluginDetailsPath := filepath.Join(outputDir, "plugins.json")
	file, err := os.Create(pluginDetailsPath) //nolint:gosec
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		slog.Error("Can't write create a plugins JSON file", "path", pluginDetailsPath)
		return
	}

	slog.Info("Plugin details file generated", "path", pluginDetailsPath)
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// parse flags
	flags := pflag.NewFlagSet("docgen", pflag.ExitOnError)
	flags.StringVar(&version, "version", "v0.0.0-dev", "version of the build")
	flags.StringVar(&outputDir, "output", "./dist/docs", "output directory")
	err := flags.Parse(os.Args[1:])
	if err != nil {
		logger.Error("Can't parse provided arguments", "err", err)
		return
	}
	// load all plugins
	plugins := []*plugin.Schema{
		builtin.Plugin(version, logger, nil),
		atlassian.Plugin(version, nil),
		crowdstrike.Plugin(version, nil),
		eclecticiq.Plugin(version, nil),
		elastic.Plugin(version, nil),
		github.Plugin(version, nil),
		graphql.Plugin(version),
		hackerone.Plugin(version, nil),
		iris.Plugin(version, nil),
		microsoft.Plugin(version, nil, nil, nil),
		misp.Plugin(version, nil),
		nistnvd.Plugin(version, nil),
		openai.Plugin(version, nil),
		opencti.Plugin(version),
		postgresql.Plugin(version),
		snyk.Plugin(version, nil),
		splunk.Plugin(version, nil),
		sqlite.Plugin(version),
		stixview.Plugin(version),
		terraform.Plugin(version),
		virustotal.Plugin(version, nil),
	}
	// generate markdown for each plugin
	for _, p := range plugins {

		log := slog.With("plugin", p.Name)

		pluginShortname := p.Shortname()

		// Use a shortname as a plugin directory name
		pluginOutputDir := filepath.Join(outputDir, pluginShortname)

		// Create a plugin directory if it doesn't exist
		err := os.MkdirAll(pluginOutputDir, 0o750)
		if err != nil {
			log.Error("Can't create a plugin directory", "path", pluginOutputDir)
			panic(err)
		}

		pluginDocPath := filepath.Join(pluginOutputDir, "_index.md")
		err = renderPluginDoc(p, pluginDocPath)
		if err != nil {
			log.Error("Error while rendering a plugin doc", "plugin", p.Name)
			panic(err)
		}

		if len(p.DataSources) != 0 {
			generateDataSourceDocs(log, p, pluginOutputDir)
		}
		if len(p.ContentProviders) != 0 {
			generateContentProviderDocs(log, p, pluginOutputDir)
		}
		if len(p.Formatters) != 0 {
			generateFormatterDocs(log, p, pluginOutputDir)
		}
		if len(p.Publishers) != 0 {
			generatePublisherDocs(log, p, pluginOutputDir)
		}

		log.Info("Plugin docs rendered", "path", pluginDocPath)

	}
	generateMetadataFile(plugins, outputDir)
}

func renderPluginDoc(pluginSchema *plugin.Schema, fp string) error {
	f, err := os.Create(fp) // nolint: gosec
	if err != nil {
		return err
	}
	defer f.Close()

	return base.ExecuteTemplate(f, "plugin", pluginSchema)
}

func renderContentProviderDoc(
	pluginSchema *plugin.Schema,
	contentProviderName string,
	contentProvider *plugin.ContentProvider,
	fp string,
) error {
	f, err := os.Create(fp) // nolint: gosec
	if err != nil {
		return err
	}
	defer f.Close()

	templContext := map[string]any{
		"plugin":           pluginSchema,
		"plugin_shortname": pluginSchema.Shortname(),
		"name":             contentProviderName,
		"content_provider": contentProvider,
		"desc":             description(contentProvider.Doc),
		"short_desc":       shortDescription(contentProvider.Doc),
	}
	return base.ExecuteTemplate(f, "content-provider", templContext)
}

func renderPublisherDoc(
	pluginSchema *plugin.Schema,
	publisherName string,
	publisher *plugin.Publisher,
	fp string,
) error {
	f, err := os.Create(fp) // nolint: gosec
	if err != nil {
		return err
	}
	defer f.Close()

	templContext := map[string]any{
		"plugin":           pluginSchema,
		"plugin_shortname": pluginSchema.Shortname(),
		"name":             publisherName,
		"publisher":        publisher,
	}
	return base.ExecuteTemplate(f, "publisher", templContext)
}

func renderFormatterDoc(
	pluginSchema *plugin.Schema,
	formatterName string,
	formatter *plugin.Formatter,
	fp string,
) error {
	f, err := os.Create(fp) // nolint: gosec
	if err != nil {
		return err
	}
	defer f.Close()

	templContext := map[string]any{
		"plugin":           pluginSchema,
		"plugin_shortname": pluginSchema.Shortname(),
		"name":             formatterName,
		"formatter":        formatter,
	}
	return base.ExecuteTemplate(f, "formatter", templContext)
}

func renderDataSourceDoc(
	pluginSchema *plugin.Schema,
	dataSourceName string,
	dataSource *plugin.DataSource,
	fp string,
) error {
	f, err := os.Create(fp) // nolint: gosec
	if err != nil {
		return err
	}
	defer f.Close()

	templContext := map[string]any{
		"plugin":           pluginSchema,
		"plugin_shortname": pluginSchema.Shortname(),
		"name":             dataSourceName,
		"data_source":      dataSource,
		"desc":             description(dataSource.Doc),
		"short_desc":       shortDescription(dataSource.Doc),
	}
	return base.ExecuteTemplate(f, "data-source", templContext)
}

func description(doc string) string {
	return utils.Dedent(doc)
}

func shortDescription(doc string) string {
	firstLine, _, _ := strings.Cut(strings.TrimSpace(doc), "\n")
	return strings.TrimRight(firstLine, ".")
}

func shortname(name string) string {
	parts := strings.SplitN(name, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return name
}

func init() {
	base = template.Must(template.New("content-provider").
		Funcs(sprig.FuncMap()).
		Funcs(template.FuncMap{
			"shortname": shortname,
			"renderDoc": dataspec.RenderDoc,
			"formatTags": func(data []string) (string, error) {
				if data == nil {
					data = []string{}
				}
				res, err := json.Marshal(data)
				return string(res), err
			},
		}).
		Parse(contentProviderTemplValue))
	template.Must(base.New("publisher").Parse(publisherTemplValue))
	template.Must(base.New("formatter").Parse(formatterTemplValue))
	template.Must(base.New("data-source").Parse(dataSourceTemplValue))
	template.Must(base.New("plugin").Parse(pluginTemplValue))
}
