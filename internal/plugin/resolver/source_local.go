package resolver

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// LocalSource is a plugin source that looks up plugins from a local directory.
// The directory structure should be:
//
//	"<path>/<namespace>/<shortname>@<version>"
//
// For example with the path ".fabric/plugins" plugin name "blackstork/sqlite" and version "1.0.0":
//
//	".fabric/plugins/blackstork/sqlite@1.0.0"
//
// File checksums can be provided in a file with the same name as the plugin binary but with a "_checksums.txt" suffix.
// The file should contain a list of checksums for all supported platforms.
type LocalSource struct {
	// path is the root directory to look up plugins.
	path   string
	tracer trace.Tracer
	logger *slog.Logger
}

// NewLocal creates a new LocalSource with the given path.
// If the logger is nil, then a new logger is created with no output.
// If the tracer is nil, then a new no-op tracer is used.
// The path should be the root directory to look up plugins.
func NewLocal(path string, logger *slog.Logger, tracer trace.Tracer) *LocalSource {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if tracer == nil {
		tracer = tracenoop.Tracer{}
	}
	logger = logger.With("source", "local").With("path", path)
	return &LocalSource{
		path:   path,
		tracer: tracer,
		logger: logger,
	}
}

// Lookup returns the versions found of the plugin with the given name.
func (source LocalSource) Lookup(ctx context.Context, name Name) (_ []Version, err error) {
	ctx, span := source.tracer.Start(ctx, "LocalSource.Lookup", trace.WithAttributes(
		attribute.String("name", name.String()),
		attribute.String("path", source.path),
	))
	source.logger.DebugContext(ctx, "Looking up plugin", "name", name)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	if source.path == "" {
		return nil, fmt.Errorf("no path provided for local source")
	}
	pluginDir := filepath.Join(source.path, name.Namespace())
	entries, err := os.ReadDir(pluginDir)
	if os.IsNotExist(err) {
		source.logger.DebugContext(ctx, "Plugins dir is not found", "name", name)
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read plugin from local dir '%s': %w", source.path, err)
	}
	var matches []Version
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		parts := strings.Split(entry.Name(), "@")
		if len(parts) != 2 {
			continue
		}
		if parts[0] != name.Short() {
			continue
		}
		parts[1] = strings.TrimSuffix(parts[1], ".exe")
		version, err := semver.NewVersion(parts[1])
		if err != nil {
			continue
		}
		matches = append(matches, Version{version})
	}
	source.logger.DebugContext(
		ctx, "Plugin versions found for a plugin name",
		"name", name,
		"matches_count", len(matches))
	return matches, nil
}

// Resolve returns the binary path and checksum for the given plugin version.
func (source LocalSource) Resolve(ctx context.Context, name Name, version Version, checksums []Checksum) (_ *ResolvedPlugin, err error) {
	ctx, span := source.tracer.Start(ctx, "LocalSource.Resolve", trace.WithAttributes(
		attribute.String("name", name.String()),
		attribute.String("version", version.String()),
		attribute.String("path", source.path),
	))
	source.logger.DebugContext(ctx, "Resolving plugin", "name", name, "version", version)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	pluginDir := filepath.Join(source.path, name.Namespace())
	pluginPath := filepath.Join(pluginDir, fmt.Sprintf("%s@%s", name.Short(), version.String()))
	checksumpath := pluginPath + "_checksums.txt"
	info, err := os.Stat(pluginPath)
	if os.IsNotExist(err) {
		info, err = os.Stat(pluginPath + ".exe")
		if os.IsNotExist(err) {
			return nil, ErrPluginNotFound
		} else if err != nil {
			return nil, fmt.Errorf("failed to stat plugin file: %w", err)
		}
		pluginPath += ".exe"
	}
	if info.IsDir() {
		return nil, fmt.Errorf("plugin file is a directory")
	}
	// calculate the checksum of the plugin binary
	h := sha256.New()
	file, err := os.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin file: %w", err)
	}
	defer file.Close()
	if _, err := io.Copy(h, file); err != nil {
		return nil, fmt.Errorf("failed to hash plugin file: %w", err)
	}
	checksum := Checksum{
		Object: "binary",
		OS:     runtime.GOOS,
		Arch:   runtime.GOARCH,
		Sum:    h.Sum(nil),
	}
	// If the checksums are not provided, then we assume the checksums are the same as the binary.
	if len(checksums) == 0 {
		// If the checksums metadata file exists then we use the checksums from the file.
		// This file is created by RemoteSource when downloading plugins.
		// This is useful when the checksums are different for different platforms.
		if f, err := os.Open(checksumpath); err == nil {
			defer f.Close()
			checksums, err = decodeChecksums(f)
			if err != nil {
				return nil, fmt.Errorf("failed to decode checksums from local source: %w", err)
			}
			// Additionally, we check that the checksums match the binary.
			if !checksum.Match(checksums) {
				return nil, fmt.Errorf("invalid plugin binary checksum: '%s'", checksum)
			}
		} else if os.IsNotExist(err) {
			// If the checksums file does not exist, then we assume the checksums are the same as the binary.
			checksums = []Checksum{checksum}
		} else {
			// If there is an error opening the checksums file, then we return the error.
			// This is useful for debugging.
			return nil, fmt.Errorf("failed to open checksums file at local source: %w", err)
		}
	} else if !checksum.Match(checksums) {
		// If the checksums are provided, then we check that the checksums match the binary.
		return nil, fmt.Errorf("invalid plugin binary checksum: '%s'", checksum)
	}
	return &ResolvedPlugin{
		BinaryPath: pluginPath,
		Checksums:  checksums,
	}, nil
}
