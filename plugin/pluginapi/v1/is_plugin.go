// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

//go:build blackstorkplugin

package pluginapiv1

import (
	"log"
	"log/slog"
	"os"
	"sync"

	"github.com/evanphx/go-hclog-slog/hclogslog"
	"github.com/hashicorp/go-hclog"
)

var logMutex sync.Mutex

func init() {
	hclog.SetDefault(hclog.New(&hclog.LoggerOptions{
		Level:                    hclog.Trace,
		Output:                   os.Stderr,
		JSONFormat:               true,
		TimeFn:                   hclog.DefaultOptions.TimeFn,
		IncludeLocation:          true,
		AdditionalLocationOffset: 0, // for direct hclog
		Mutex:                    &logMutex,
	}))

	// Make slog use hclogger
	// NOTE: slog.SetDefault also calls log.SetOutput, the order of operations is important
	slog.SetDefault(slog.New(hclogslog.Adapt(hclog.New(&hclog.LoggerOptions{
		Level:                    hclog.Trace,
		Output:                   os.Stderr,
		JSONFormat:               true,
		TimeFn:                   hclog.DefaultOptions.TimeFn,
		IncludeLocation:          true,
		AdditionalLocationOffset: 3, // for slog
		Mutex:                    &logMutex,
	}))))

	// Make standard logger use hclogger
	log.SetOutput(hclog.Default().StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	}))
	log.SetPrefix("")
	log.SetFlags(0)
}

func loggerForGoplugin() hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Level:                    hclog.Info, // Debug is too verbose
		Output:                   os.Stderr,
		JSONFormat:               true,
		TimeFn:                   hclog.DefaultOptions.TimeFn,
		IncludeLocation:          true,
		AdditionalLocationOffset: 0, // for direct hclog
		Mutex:                    &logMutex,
	})
}
