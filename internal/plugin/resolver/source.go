package resolver

import (
	"context"
	"fmt"
	"slices"
)

// ErrPluginNotFound is returned when a plugin is not found in the source.
var ErrPluginNotFound = fmt.Errorf("plugin not found")

// ResolvedPlugin contains the binary path and checksum for a plugin.
type ResolvedPlugin struct {
	// BinaryPath for the current platform.
	BinaryPath string
	// Checksums is a list of checksums for the plugin including all supported platforms.
	Checksums []Checksum
}

// Source is the interface for plugin sources.
// A source is responsible for listing, looking up, and resolving plugins.
// The source may use a local directory, a registry, or any other source.
// If the source is unable to find a plugin, it should return ErrPluginNotFound.
type Source interface {
	// Lookup returns a list of available versions for the given plugin.
	Lookup(ctx context.Context, name Name) ([]Version, error)
	// Resolve returns the binary path and checksums for the given plugin version.
	Resolve(ctx context.Context, name Name, version Version, checksums []Checksum) (*ResolvedPlugin, error)
}

// makeSourceChain returns a source that chains the given sources together.
// When looking up a plugin, the sources are queried in order and the results are concatenated and sorted.
// When resolving a plugin, the sources are queried in order and the first result is returned.
// If a source returns an error other than ErrPluginNotFound, the chain is interrupted and the error is returned.
// If all sources return ErrPluginNotFound, then ErrPluginNotFound is returned.
func makeSourceChain(sources ...Source) Source {
	return &sourceChain{sources}
}

type sourceChain struct {
	sources []Source
}

func (source *sourceChain) Lookup(ctx context.Context, name Name) ([]Version, error) {
	var matches []Version
	for _, s := range source.sources {
		found, err := s.Lookup(ctx, name)
		if err != nil && err != ErrPluginNotFound {
			return nil, err
		}
		matches = append(matches, found...)
	}
	slices.SortFunc(matches, func(a, b Version) int {
		return a.Compare(b)
	})
	return slices.Compact(matches), nil
}

func (source *sourceChain) Resolve(ctx context.Context, name Name, version Version, checksums []Checksum) (*ResolvedPlugin, error) {
	for _, s := range source.sources {
		info, err := s.Resolve(ctx, name, version, checksums)
		if err == nil {
			return info, nil
		}
		if err != ErrPluginNotFound {
			return nil, err
		}
	}
	return nil, ErrPluginNotFound
}
