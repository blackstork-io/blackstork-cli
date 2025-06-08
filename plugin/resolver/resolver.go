package resolver

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/hashicorp/hcl/v2"
	"go.opentelemetry.io/otel/codes"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
)

// Resolver resolves and installs plugins.
type Resolver struct {
	constraints ConstraintMap
	options
}

// NewResolver creates a new plugin resolver.
func NewResolver(constraints map[string]string, opts ...Option) (*Resolver, diagnostics.Diag) {
	parsedVersions, err := ParseConstraintMap(constraints)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse plugin versions",
			Detail:   err.Error(),
		}}
	}
	res := &Resolver{
		constraints: parsedVersions,
		options:     defaultOptions,
	}
	for _, opt := range opts {
		opt(&res.options)
	}
	res.options.logger = res.options.logger.With("component", "resolver")
	return res, nil
}

// Install all plugins based the version constraints and return updated a lock file.
func (r *Resolver) Install(ctx context.Context, lockFile *LockFile, upgrade bool) (_ *LockFile, diags diagnostics.Diag) {
	ctx, span := r.tracer.Start(ctx, "Resolver.Install")
	r.logger.InfoContext(ctx, "Resolving and installing plugin dependencies", "upgrade", upgrade)
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
		}
		span.End()
	}()
	check := lockFile.Check(r.constraints)
	locks := []PluginLock{}
	// lookupMap is a map of plugins that are we look up based on the constraints
	lookupMap := make(ConstraintMap)
	if upgrade {
		// if upgrade is enabled we install all plugins based on the constraints
		maps.Copy(lookupMap, r.constraints)
	} else {
		// if upgrade is disabled we only install the missing and mismatched plugins based on the constraints
		maps.Copy(lookupMap, check.Missing)
		maps.Copy(lookupMap, check.Mismatch)
	}
	chain := makeSourceChain(r.sources...)
	// resolve the plugins by the latest version that matches the constraints
	for name, constraint := range lookupMap {
		r.logger.InfoContext(ctx, "Looking for a plugin", "name", name.String(), "constraints", constraint.String())
		list, err := chain.Lookup(ctx, name)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Failed to find plugin '%s'", name),
				Detail:   err.Error(),
			}}
		}
		if len(list) == 0 {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Plugin '%s' not found", name),
				Detail:   "Can't find requested plugin for the current platform",
			}}
		}
		// filter out the versions that do not match the constraint
		matches := slices.DeleteFunc(list, func(v Version) bool {
			return !constraint.Check(v.Version)
		})
		if len(matches) == 0 {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Plugin '%s' not found", name),
				Detail:   fmt.Sprintf("No version of '%s' matches the constraint '%s'", name, constraint),
			}}
		}
		max := slices.MaxFunc(matches, func(a, b Version) int {
			return a.Compare(b)
		})
		r.logger.InfoContext(ctx, "Installing the plugin", "name", name.String(), "version", max.String())
		var checksums []Checksum
		// check if the plugin with the same version is already in the lock file
		lockIdx := slices.IndexFunc(lockFile.Plugins, func(lock PluginLock) bool {
			return lock.Name == name && lock.Version.Equal(max.Version)
		})
		if lockIdx > -1 {
			// use the checksums from the lock file
			checksums = lockFile.Plugins[lockIdx].Checksums
		}
		res, err := chain.Resolve(ctx, name, max, checksums)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Failed to resolve plugin '%s@%s'", name, max),
				Detail:   err.Error(),
			}}
		}
		// sort the checksums
		slices.SortFunc(res.Checksums, func(a, b Checksum) int {
			return a.Compare(b)
		})
		locks = append(locks, PluginLock{
			Name:      name,
			Version:   max,
			Checksums: res.Checksums,
		})
		// check if context is cancelled
		if ctx.Err() != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Cancelled by context",
				Detail:   ctx.Err().Error(),
			}}
		}
	}
	// resolve the rest of plugins based on the strict locked versions
	for _, lock := range lockFile.Plugins {
		// skip plugins that are already resolved
		if _, ok := lookupMap[lock.Name]; ok {
			continue
		}
		// skip plugins that are removed from the version constraints
		if _, ok := check.Removed[lock.Name]; ok {
			continue
		}
		r.logger.InfoContext(ctx, "Installing the plugin", "name", lock.Name.String(), "version", lock.Version.String())
		_, err := chain.Resolve(ctx, lock.Name, lock.Version, lock.Checksums)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Failed to resolve plugin '%s@%s'", lock.Name, lock.Version),
				Detail:   err.Error(),
			}}
		}
		locks = append(locks, lock)
		// check if context is cancelled
		if ctx.Err() != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Cancelled by context",
				Detail:   ctx.Err().Error(),
			}}
		}
	}
	// sort plugin locks by name
	slices.SortFunc(locks, func(a, b PluginLock) int {
		return a.Name.Compare(b.Name)
	})
	return &LockFile{
		Plugins: locks,
	}, nil
}

// Resolve all plugins based on the constraints and returns a map of plugin names to binary paths.
// If the lock file is not satisfied, an error is returned.
func (r *Resolver) Resolve(ctx context.Context, lockFile *LockFile) (_ map[string]string, diags diagnostics.Diag) {
	ctx, span := r.tracer.Start(ctx, "Resolver.Resolve")
	r.logger.DebugContext(ctx, "Resolving plugins", "lock_file_plugins_count", len(lockFile.Plugins))
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	// check if the lock file is satisfied by version constraints
	check := lockFile.Check(r.constraints)
	for name, lock := range check.Removed {
		// warn about plugins that are removed from the version constraints
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  fmt.Sprintf("Plugin `%s` is not used", name),
			Detail:   fmt.Sprintf("Version `%s` is not in use. To update the lock file, run `install` command", lock),
		})
	}
	if check.IsInstallRequired() {
		// error out about missing & mismatched plugins
		for name := range check.Missing {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Plugin `%s` is not locked", name),
				Detail:   "Run install to resolve missing plugins.",
			})
		}
		for name, constraint := range check.Mismatch {
			pluginIdx := slices.IndexFunc(lockFile.Plugins, func(lock PluginLock) bool {
				return lock.Name == name
			})
			if pluginIdx == -1 {
				continue
			}
			detailFormat := "Version locked at `%s` does not match the new constraint `%s`. To update the lock file, run `install` command."
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Plugin `%s` version mismatch", name),
				Detail:   fmt.Sprintf(detailFormat, lockFile.Plugins[pluginIdx].Version, constraint),
			})
		}
		return nil, diags
	}
	// chain the sources together
	chain := makeSourceChain(r.sources...)
	// resolve the plugins
	binaryMap := make(map[string]string)
	for _, lock := range lockFile.Plugins {
		// skip plugins that are removed from the version constraints
		if _, ok := check.Removed[lock.Name]; ok {
			continue
		}
		plugin, err := chain.Resolve(ctx, lock.Name, lock.Version, lock.Checksums)
		if err != nil {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Failed to resolve plugin `%s@%s`", lock.Name, lock.Version),
				Detail:   err.Error(),
			})
			return nil, diags
		}
		binaryMap[lock.Name.String()] = plugin.BinaryPath
		// check if context is cancelled
		if ctx.Err() != nil {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Context cancelled",
				Detail:   ctx.Err().Error(),
			})
			return nil, diags
		}
	}
	return binaryMap, diags
}
