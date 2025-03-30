package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

var (
	version    string
	namespace  string
	output     string
	configFile string
	osName     string
	archName   string
	plugin     string
)

// This is used to generate plugin metadata for release
func main() {
	flags := pflag.NewFlagSet("pluginmeta", pflag.ExitOnError)
	flags.StringVar(&namespace, "namespace", "blackstork", "namespace for plugins")
	flags.StringVar(&configFile, "config", ".goreleaser.yaml", "path to goreleaser config")
	flags.StringVar(&output, "output", ".tmp/plugins.json", "path to output plugins.json")
	flags.StringVar(&version, "version", "0.0.0", "version for plugins")
	flags.StringVar(&osName, "os", "", "os for patch")
	flags.StringVar(&archName, "arch", "", "arch for patch")
	flags.StringVar(&plugin, "plugin", "", "plugin for patch")
	if err := flags.Parse(os.Args[1:]); err != nil {
		panic(err)
	}
	args := flags.Args()
	if len(args) == 1 && args[0] == "patch" {
		// Patch metadata
		meta, err := readMeta()
		if err != nil {
			panic(err)
		}
		err = patchMeta(meta, plugin, osName, archName)
		if err != nil {
			panic(err)
		}
		return
	}
	// Read and parse config
	cfg, err := readConfig()
	if err != nil {
		panic(err)
	}
	meta, err := parseConfig(cfg)
	if err != nil {
		panic(err)
	}
	// Write metadata
	err = os.MkdirAll(filepath.Dir(output), 0o750)
	if err != nil {
		panic(err)
	}
	err = writeMetadata(meta)
	if err != nil {
		panic(err)
	}
}

func patchMeta(meta *Metadata, plugin, osName, archName string) error {
	split := strings.Split(filepath.Base(plugin), "@")
	if len(split) != 2 {
		return fmt.Errorf("invalid plugin name")
	}
	name := split[0]
	var archive *PluginArchiveMetadata
	for _, p := range meta.Plugins {
		if p.Name != fmt.Sprintf("%s/%s", namespace, name) {
			continue
		}
		for _, a := range p.Archives {
			if a.OS != osName || a.Arch != archName {
				continue
			}
			archive = a
			break
		}
		break
	}
	if archive == nil {
		return fmt.Errorf("archive not found")
	}
	f, err := os.Open(plugin) //nolint:gosec // The plugin path comes from flags and is controlled by admin
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return err
	}
	archive.BinaryChecksum = base64.StdEncoding.EncodeToString(h.Sum(nil))
	return writeMetadata(meta)
}

func readMeta() (*Metadata, error) {
	f, err := os.Open(output) //nolint:gosec // Output path is controlled by admin configuration
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var meta Metadata
	err = json.NewDecoder(f).Decode(&meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func readConfig() (*ReleaserConfig, error) {
	f, err := os.Open(configFile) //nolint:gosec // Config file path is controlled by admin configuration
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var config ReleaserConfig
	err = yaml.NewDecoder(f).Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// parseConfig creates metadata for plugins from the given goreleaser configuration.
func parseConfig(cfg *ReleaserConfig) (*Metadata, error) {
	plugins := make([]*PluginMetadata, 0)
	for _, artifact := range cfg.Archives {
		if !strings.HasPrefix(artifact.ID, "plugin_") {
			continue
		}
		if len(artifact.Builds) != 1 {
			return nil, fmt.Errorf("plugin artifacts must have exactly one build")
		}
		buildIdx := slices.IndexFunc(cfg.Builds, func(b ReleaserBuild) bool {
			return b.ID == artifact.Builds[0]
		})
		if buildIdx == -1 {
			return nil, fmt.Errorf("build not found")
		}
		build := cfg.Builds[buildIdx]
		if len(build.GOOS) == 0 {
			return nil, fmt.Errorf("build must have at least one GOOS")
		}
		plugin := &PluginMetadata{
			Name:     namespace + "/" + strings.TrimPrefix(artifact.ID, "plugin_"),
			Version:  version,
			Archives: make([]*PluginArchiveMetadata, 0),
		}
		tmpl := template.Must(template.New("name").Parse(artifact.NameTemplate))
		for _, goos := range build.GOOS {
			archList := osArchList(goos)
			ext := artifact.Format
			for _, arch := range archList {
				args := map[string]any{
					"Os":   goos,
					"Arch": arch,
					"Arm":  nil,
				}
				var filename strings.Builder
				err := tmpl.Execute(&filename, args)
				if err != nil {
					return nil, err
				}
				binary := &PluginArchiveMetadata{
					Filename: filename.String() + "." + ext,
					OS:       goos,
					Arch:     arch,
				}
				plugin.Archives = append(plugin.Archives, binary)
			}
		}
		plugins = append(plugins, plugin)
	}
	return &Metadata{Plugins: plugins}, nil
}

func osArchList(goos string) []string {
	switch goos {
	case "linux":
		return []string{"amd64", "arm64", "386"}
	case "darwin":
		return []string{"amd64", "arm64"}
	case "windows":
		return []string{"amd64", "386", "arm64"}
	default:
		return []string{}
	}
}

func writeMetadata(metadata *Metadata) error {
	f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) //nolint:gosec // Output path is controlled by admin configuration
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(metadata)
	if err != nil {
		return err
	}
	return nil
}
