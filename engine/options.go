// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package engine

import (
	"io"
	"log/slog"

	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugins/builtin"
)

const (
	defaultRegistryBaseURL = "https://registry.blackstork.io"
	defaultCacheDir        = ".blackstork"
	defaultLockFile        = ".blackstork-lock.json"
)

// Options is a set of options for the engine.
type Options struct {
	registryBaseURL string
	cacheDir        string
	builtin         *plugin.Schema
}

var defaultLogger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
	Level: slog.LevelError,
}))

var defaultOptions = Options{
	registryBaseURL: defaultRegistryBaseURL,
	cacheDir:        defaultCacheDir,
	builtin:         builtin.Plugin("v0.0.0", defaultLogger, nil),
}

type Option func(*Options)

// WithRegistryBaseURL sets the registry base URL. Default is "https://registry.blackstork.io".
func WithRegistryBaseURL(url string) Option {
	return func(o *Options) {
		o.registryBaseURL = url
	}
}

// WithCacheDir sets the cache directory. Default is ".fabric".
func WithCacheDir(dir string) Option {
	return func(o *Options) {
		o.cacheDir = dir
	}
}

// WithBuiltIn sets the built-in plugin.
func WithBuiltIn(builtin *plugin.Schema) Option {
	return func(o *Options) {
		o.builtin = builtin
	}
}
