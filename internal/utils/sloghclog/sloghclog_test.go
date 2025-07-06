// Transform slog.Logger into hclog.Logger instance
package sloghclog

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

var testTime = time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC)

func replaceTime(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		a = slog.Time(a.Key, testTime)
	}
	return a
}

func Test_levelConversion(t *testing.T) {
	t.Parallel()
	// levelTrace should continue slog level system, but below the debug
	assert.Less(t, LevelTrace, slog.LevelDebug)
	assert.Equal(t, slog.LevelInfo-slog.LevelDebug, slog.LevelDebug-LevelTrace)

	tests := []struct {
		hclog hclog.Level
		slog  slog.Level
	}{
		{hclog.Trace, LevelTrace},
		{hclog.Debug, slog.LevelDebug},
		{hclog.Info, slog.LevelInfo},
		{hclog.DefaultLevel, slog.LevelInfo},
		{hclog.Warn, slog.LevelWarn},
		{hclog.Error, slog.LevelError},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.slog, levelToSlog(tt.hclog))
		assert.Equal(t, tt.hclog, levelToHclog(tt.slog))
	}
	assert.Equal(t, slog.LevelInfo, levelToSlog(hclog.NoLevel))
	assert.NotEqual(t, hclog.NoLevel, levelToHclog(slog.Level(-12)))
}

func Test_isLevel(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	levels := []slog.Level{
		LevelTrace,
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
	}
	for _, logLevel := range levels {
		a := Adapt(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: logLevel,
		})))
		assert.Equal(logLevel <= LevelTrace, a.IsTrace())
		assert.Equal(logLevel <= slog.LevelDebug, a.IsDebug())
		assert.Equal(logLevel <= slog.LevelInfo, a.IsInfo())
		assert.Equal(logLevel <= slog.LevelWarn, a.IsWarn())
		assert.Equal(logLevel <= slog.LevelError, a.IsError())
	}
}

func Test_levelRespected(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	buf := &bytes.Buffer{}

	for _, logLevel := range []slog.Level{slog.LevelDebug, slog.LevelInfo} {
		for _, adapterLevel := range []slog.Level{minLevel, slog.LevelDebug, slog.LevelInfo} {
			logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
				Level: logLevel,
			}))
			var a hclog.Logger
			if adapterLevel == minLevel {
				a = Adapt(logger)
			} else {
				a = Adapt(logger, Level(adapterLevel))
			}
			for _, msgLevel := range []hclog.Level{hclog.Debug, hclog.Info} {
				slogMsgLevel := levelToSlog(msgLevel)
				a.Log(msgLevel, "test")
				msg := readBuf(buf)
				if logLevel <= slogMsgLevel && adapterLevel <= slogMsgLevel {
					assert.NotEmpty(msg)
				} else {
					assert.Empty(msg)
				}
			}
		}
	}

	levels := []slog.Level{
		LevelTrace,
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
	}
	for _, logLevel := range levels {
		a := Adapt(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: logLevel,
		})))
		assert.Equal(logLevel <= LevelTrace, a.IsTrace())
		assert.Equal(logLevel <= slog.LevelDebug, a.IsDebug())
		assert.Equal(logLevel <= slog.LevelInfo, a.IsInfo())
		assert.Equal(logLevel <= slog.LevelWarn, a.IsWarn())
		assert.Equal(logLevel <= slog.LevelError, a.IsError())
	}
}

func Test_levelSettable(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	buf := &bytes.Buffer{}

	for _, logLevel := range []slog.Level{slog.LevelDebug, slog.LevelInfo} {
		a := Adapt(
			slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
				Level: logLevel,
			})),
			Level(&slog.LevelVar{}),
		)
		for _, adapterLevel := range []hclog.Level{hclog.Trace, hclog.Debug, hclog.Info} {
			a.SetLevel(adapterLevel)
			for _, msgLevel := range []hclog.Level{hclog.Trace, hclog.Debug, hclog.Info} {
				slogMsgLevel := levelToSlog(msgLevel)
				a.Log(msgLevel, "test")
				msg := readBuf(buf)
				if logLevel <= slogMsgLevel && adapterLevel <= msgLevel {
					assert.NotEmpty(msg)
				} else {
					assert.Empty(msg)
				}
			}
		}

	}

	levels := []slog.Level{
		LevelTrace,
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
	}
	for _, logLevel := range levels {
		a := Adapt(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: logLevel,
		})))
		assert.Equal(logLevel <= LevelTrace, a.IsTrace())
		assert.Equal(logLevel <= slog.LevelDebug, a.IsDebug())
		assert.Equal(logLevel <= slog.LevelInfo, a.IsInfo())
		assert.Equal(logLevel <= slog.LevelWarn, a.IsWarn())
		assert.Equal(logLevel <= slog.LevelError, a.IsError())
	}
}

func readBuf(buf *bytes.Buffer) string {
	return string(buf.Next(buf.Len()))
}

func Test_adapterName(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	buf := &bytes.Buffer{}

	a := Adapt(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: replaceTime,
	})))
	assert.Equal("", a.Name())
	a.Info("test")
	assert.Equal(
		"time=2000-01-02T03:04:05.000Z level=INFO msg=test\n",
		readBuf(buf),
	)

	a = a.Named("1")
	assert.Equal("1", a.Name())
	a.Info("test")
	assert.Equal(
		"time=2000-01-02T03:04:05.000Z level=INFO msg=test name=1\n",
		readBuf(buf),
	)

	a = a.Named("2")
	assert.Equal("1.2", a.Name())
	a.Info("test")
	assert.Equal(
		"time=2000-01-02T03:04:05.000Z level=INFO msg=test name=1.2\n",
		readBuf(buf),
	)

	a = a.ResetNamed("3")
	assert.Equal("3", a.Name())
	a.Info("test")
	assert.Equal(
		"time=2000-01-02T03:04:05.000Z level=INFO msg=test name=3\n",
		readBuf(buf),
	)

	a = a.ResetNamed("")
	assert.Equal("", a.Name())
	a.Info("test")
	assert.Equal(
		"time=2000-01-02T03:04:05.000Z level=INFO msg=test\n",
		readBuf(buf),
	)
}

func Test_adapterIndependance(t *testing.T) {
	t.Parallel()
	t.Run("calling named funcs doesn't affect the parent", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		buf := &bytes.Buffer{}

		a := Adapt(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
			ReplaceAttr: replaceTime,
		})))

		a2 := a.Named("somename")
		assert.Equal("somename", a2.Name())
		assert.Equal("", a.Name())

		a2.Info("test_a2")
		assert.Equal(
			"time=2000-01-02T03:04:05.000Z level=INFO msg=test_a2 name=somename\n",
			readBuf(buf),
		)

		a.Info("test_a")
		assert.Equal(
			"time=2000-01-02T03:04:05.000Z level=INFO msg=test_a\n",
			readBuf(buf),
		)
	})

	t.Run("calling with funcs doesn't affect the parent", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		buf := &bytes.Buffer{}

		a := Adapt(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
			ReplaceAttr: replaceTime,
		})))
		a1 := a.With()
		a2 := a1.With("someattr", "someval")

		assert.Empty(a.ImpliedArgs())
		assert.Empty(a1.ImpliedArgs())
		assert.Exactly([]any{"someattr", "someval"}, a2.ImpliedArgs())

		a.Info("test_a")
		assert.Equal(
			"time=2000-01-02T03:04:05.000Z level=INFO msg=test_a\n",
			readBuf(buf),
		)

		a1.Info("test_a1")
		assert.Equal(
			"time=2000-01-02T03:04:05.000Z level=INFO msg=test_a1\n",
			readBuf(buf),
		)

		a2.Info("test_a2")
		assert.Equal(
			"time=2000-01-02T03:04:05.000Z level=INFO msg=test_a2 someattr=someval\n",
			readBuf(buf),
		)
	})

	t.Run("named and with interaction", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		buf := &bytes.Buffer{}

		a := Adapt(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
			Level:       LevelTrace,
			ReplaceAttr: replaceTime,
		})))

		a2 := a.With("someattr", "someval")
		a3 := a2.Named("a3_named")
		a4 := a3.Named("a4_named")
		a5 := a4.With("nextattr", "nextval")
		a6 := a5.ResetNamed("")

		a.Trace("test_a")
		assert.Equal(
			"time=2000-01-02T03:04:05.000Z level=DEBUG-4 msg=test_a\n",
			readBuf(buf),
		)

		a2.Debug("test_a2")
		assert.Equal(
			"time=2000-01-02T03:04:05.000Z level=DEBUG msg=test_a2 someattr=someval\n",
			readBuf(buf),
		)

		a3.Info("test_a3")
		assert.Equal(
			"time=2000-01-02T03:04:05.000Z level=INFO msg=test_a3 name=a3_named someattr=someval\n",
			readBuf(buf),
		)

		a4.Warn("test_a4")
		assert.Equal(
			"time=2000-01-02T03:04:05.000Z level=WARN msg=test_a4 name=a3_named.a4_named someattr=someval\n",
			readBuf(buf),
		)

		a5.Error("test_a5")
		assert.Equal(
			"time=2000-01-02T03:04:05.000Z level=ERROR msg=test_a5 name=a3_named.a4_named someattr=someval nextattr=nextval\n",
			readBuf(buf),
		)

		a6.Info("test_a6")
		assert.Equal(
			"time=2000-01-02T03:04:05.000Z level=INFO msg=test_a6 someattr=someval nextattr=nextval\n",
			readBuf(buf),
		)

		assert.EqualValues(a5.ImpliedArgs(), a6.ImpliedArgs())
		assert.EqualValues([]any{"someattr", "someval", "nextattr", "nextval"}, a6.ImpliedArgs())
	})
}

func Test_adapterOpts(t *testing.T) {
	t.Parallel()
	t.Run("Option Name", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		a := Adapt(
			slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
				ReplaceAttr: replaceTime,
			})),
			Name("somename"),
		)
		a.Info("test")

		assert.Equal(
			t,
			"time=2000-01-02T03:04:05.000Z level=INFO msg=test name=somename\n",
			readBuf(buf),
		)
	})
	t.Run("Option AddSource Enabled", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		buf := &bytes.Buffer{}

		a := Adapt(
			slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
				AddSource:   true,
				ReplaceAttr: replaceTime,
			})),
			AddSource(true),
		)
		a.Info("test")
		msg := readBuf(buf)

		assert.True(strings.HasPrefix(
			msg,
			"time=2000-01-02T03:04:05.000Z level=INFO source=",
		), msg)
		assert.Contains(
			msg,
			"sloghclog_test.go:",
		)
		assert.True(strings.HasSuffix(
			msg,
			" msg=test\n",
		), msg)
	})
	t.Run("Option AddSource Disabled", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		a := Adapt(
			slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
				AddSource:   true,
				ReplaceAttr: replaceTime,
			})),
			AddSource(false), // default
		)
		a.Info("test")

		assert.Equal(
			t,
			"time=2000-01-02T03:04:05.000Z level=INFO source=:0 msg=test\n",
			readBuf(buf),
		)
	})
}
