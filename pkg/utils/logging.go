// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package utils

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"golang.org/x/exp/maps"

	"github.com/blackstork-io/blackstork-cli/pkg/utils/multilog"
)

var levelNameToEnum = map[string]log.Level{
	"debug": log.DebugLevel,
	"info":  log.InfoLevel,
	"warn":  log.WarnLevel,
	"error": log.ErrorLevel,
}

func GetLogLevelsString() string {
	levelNames := maps.Keys(levelNameToEnum)
	return strings.Join(levelNames, ", ")
}

func ConfigureLogging(version, levelName, outputFormat string, otelEnabled bool) error {
	level, ok := levelNameToEnum[levelName]
	if !ok {
		return fmt.Errorf("unknown logging level: %s", levelName)
	}

	outputName := strings.ToLower(strings.TrimSpace(outputFormat))

	var handler slog.Handler

	switch outputName {
	case "plain":
		_handler := log.New(os.Stderr)
		_handler.SetReportTimestamp(true)
		_handler.SetLevel(level)
		_handler.SetReportCaller(true)
		handler = _handler
	case "json":
		opts := &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		}
		handler = slog.NewJSONHandler(os.Stderr, opts)
	default:
		return fmt.Errorf("unknown log output format '%s'", outputFormat) //nolint: err113
	}

	if otelEnabled {
		handler = multilog.Handler{
			Level: slog.Level(level),
			Handlers: []slog.Handler{
				handler,
				otelslog.NewHandler(
					"github.com/blackstork-io/blackstork-cli",
					otelslog.WithVersion(version),
				),
			},
		}
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(slog.LevelDebug) // Catch all

	slog.Info("Logging configured", "level", levelName)

	return nil
}
