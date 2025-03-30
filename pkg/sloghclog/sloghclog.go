// Transform slog.Logger into hclog.Logger instance
package sloghclog

import (
	"context"
	"io"
	"log"
	"log/slog"
	"math"
	"runtime"
	"time"

	"github.com/hashicorp/go-hclog"
)

const (
	LevelTrace slog.Level = slog.Level(-8)
	minLevel              = slog.Level(math.MinInt)
)

type adapter struct {
	origLogger *slog.Logger
	logger     *slog.Logger
	name       string
	with       []any
	addSource  bool
	leveler    slog.Leveler
}

func Adapt(logger *slog.Logger, opts ...adapterOption) hclog.Logger {
	a := &adapter{
		origLogger: logger,
		logger:     logger,
		leveler:    minLevel,
	}
	for _, opt := range opts {
		opt.apply(a)
	}
	return a
}

type adapterOption interface {
	apply(a *adapter)
}

type funcAdapterOption func(a *adapter)

func (fao funcAdapterOption) apply(a *adapter) {
	fao(a)
}

func AddSource(val bool) adapterOption {
	return funcAdapterOption(func(a *adapter) {
		a.addSource = val
	})
}

func Name(val string) adapterOption {
	return funcAdapterOption(func(a *adapter) {
		a.name = val
		a.applyName()
	})
}

func Level(level slog.Leveler) adapterOption {
	return funcAdapterOption(func(a *adapter) {
		a.leveler = level
	})
}

func levelToSlog(level hclog.Level) slog.Level {
	if level == hclog.NoLevel {
		return slog.LevelInfo
	}
	return slog.Level((level - 3) * 4)
}

func levelToHclog(level slog.Level) (res hclog.Level) {
	// Safely convert level/4 + 3 to hclog.Level, ensuring no overflow
	levelValue := int64(level) / 4
	if levelValue > math.MaxInt32-3 {
		levelValue = math.MaxInt32 - 3
	}
	res = hclog.Level(levelValue + 3) //nolint:gosec // overflow is handled above
	if res == hclog.NoLevel {
		res--
	}
	return res
}

// Args are alternating key, val pairs
// keys must be strings
// vals can be any type, but display is implementation specific
// Emit a message and key/value pairs at a provided log level
func (a *adapter) Log(level hclog.Level, msg string, args ...interface{}) {
	a.log(level, msg, args...)
}

func (a *adapter) enabled(level slog.Level) bool {
	return a.leveler.Level() <= level && a.logger.Enabled(context.Background(), level)
}

func (a *adapter) log(level hclog.Level, msg string, args ...interface{}) {
	lvl := levelToSlog(level)
	if !a.enabled(lvl) {
		return
	}
	var pcs [1]uintptr
	if a.addSource {
		// skip [Callers, log, (Log|Info|...)]
		runtime.Callers(3, pcs[:])
	}
	r := slog.NewRecord(time.Now(), lvl, msg, pcs[0])
	r.Add(args...)
	_ = a.logger.Handler().Handle(context.Background(), r)
}

// Emit a message and key/value pairs at the TRACE level
func (a *adapter) Trace(msg string, args ...interface{}) {
	a.log(hclog.Trace, msg, args...)
}

// Emit a message and key/value pairs at the DEBUG level
func (a *adapter) Debug(msg string, args ...interface{}) {
	a.log(hclog.Debug, msg, args...)
}

// Emit a message and key/value pairs at the INFO level
func (a *adapter) Info(msg string, args ...interface{}) {
	a.log(hclog.Info, msg, args...)
}

// Emit a message and key/value pairs at the WARN level
func (a *adapter) Warn(msg string, args ...interface{}) {
	a.log(hclog.Warn, msg, args...)
}

// Emit a message and key/value pairs at the ERROR level
func (a *adapter) Error(msg string, args ...interface{}) {
	a.log(hclog.Error, msg, args...)
}

// Indicate if TRACE logs would be emitted. This and the other Is* guards
// are used to elide expensive logging code based on the current level.
func (a *adapter) IsTrace() bool {
	return a.enabled(LevelTrace)
}

// Indicate if DEBUG logs would be emitted. This and the other Is* guards
func (a *adapter) IsDebug() bool {
	return a.enabled(slog.LevelDebug)
}

// Indicate if INFO logs would be emitted. This and the other Is* guards
func (a *adapter) IsInfo() bool {
	return a.enabled(slog.LevelInfo)
}

// Indicate if WARN logs would be emitted. This and the other Is* guards
func (a *adapter) IsWarn() bool {
	return a.enabled(slog.LevelWarn)
}

// Indicate if ERROR logs would be emitted. This and the other Is* guards
func (a *adapter) IsError() bool {
	return a.enabled(slog.LevelError)
}

// ImpliedArgs returns With key/value pairs
func (a *adapter) ImpliedArgs() []interface{} {
	return a.with
}

// Creates a sublogger that will always have the given key/value pairs
func (a *adapter) With(args ...interface{}) hclog.Logger {
	dup := *a

	length := len(a.with) + len(args)
	if a.name != "" {
		length += 2
	}

	with := make([]any, 0, length)
	if a.name != "" {
		with = append(with, "name", a.name)
	}
	with = append(with, a.with...)
	with = append(with, args...)

	if a.name != "" {
		dup.with = with[2:]
	} else {
		dup.with = with
	}

	if len(with) != 0 {
		dup.logger = a.origLogger.With(with...)
	} else {
		dup.logger = a.origLogger
	}
	return &dup
}

func (a *adapter) applyName() {
	var with []any
	if a.name != "" {
		with = make([]any, 0, len(a.with)+2)
		with = append(with, "name", a.name)
		with = append(with, a.with...)
	} else {
		with = a.with
	}
	if len(with) != 0 {
		a.logger = a.origLogger.With(with...)
	} else {
		a.logger = a.origLogger
	}
}

// Returns the Name of the logger
func (a *adapter) Name() string {
	return a.name
}

// Create a logger that will prepend the name string on the front of all messages.
// If the logger already has a name, the new value will be appended to the current
// name. That way, a major subsystem can use this to decorate all it's own logs
// without losing context.
func (a *adapter) Named(name string) hclog.Logger {
	dup := *a
	if dup.name != "" {
		dup.name = dup.name + "." + name
	} else {
		dup.name = name
	}
	dup.applyName()
	return &dup
}

// Create a logger that will prepend the name string on the front of all messages.
// This sets the name of the logger to the value directly, unlike Named which honor
// the current name as well.
func (a *adapter) ResetNamed(name string) hclog.Logger {
	dup := *a
	dup.name = name
	dup.applyName()
	return &dup
}

// Updates the level. This should affect all sub-loggers as well. If an
// implementation cannot update the level on the fly, it should no-op.
func (a *adapter) SetLevel(level hclog.Level) {
	type settableLeveler interface {
		Set(l slog.Level)
	}
	setter, ok := a.leveler.(settableLeveler)
	if ok {
		setter.Set(levelToSlog(level))
	}
}

// Returns the current level
func (a *adapter) GetLevel() hclog.Level {
	return levelToHclog(a.leveler.Level())
}

// Return a value that conforms to the stdlib log.Logger interface
func (a *adapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	if opts == nil {
		opts = &hclog.StandardLoggerOptions{}
	}
	// cant infer levels

	return log.New(a.StandardWriter(opts), "", 0)
}

// Return a value that conforms to io.Writer, which can be passed into log.SetOutput()
func (a *adapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return &stdlogAdapter{
		log:   a,
		level: levelToSlog(opts.ForceLevel),
	}
}

type stdlogAdapter struct {
	log   *adapter
	level slog.Level
}

// Write implements io.Writer.
func (sa *stdlogAdapter) Write(data []byte) (n int, err error) {
	sa.log.logger.Log(context.Background(), sa.level, string(data))
	return len(data), nil
}
