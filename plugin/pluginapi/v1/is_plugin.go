//go:build fabricplugin

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
