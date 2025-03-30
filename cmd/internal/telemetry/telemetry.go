package telemetry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/exp/slog"
)

// SetupOtelp bootstraps the OpenTelemetry SDK with the OTLP exporters.
func SetupOtelp(ctx context.Context, otelpURL, version string) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	prop := makePropagator()
	otel.SetTextMapPropagator(prop)

	resource, err := makeResource(ctx, version)
	if err != nil {
		return nil, err
	}
	tracerProvider, err := makeOtelpTracerProvider(ctx, resource, otelpURL)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	meterProvider, err := makeOtelpMeterProvider(ctx, resource)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)
	logProvider, err := makeOtelpLoggerProvider(ctx, resource)
	if err != nil {
		handleErr(err)
		return
	}
	global.SetLoggerProvider(logProvider)
	shutdownFuncs = append(shutdownFuncs, logProvider.Shutdown)
	err = host.Start()
	if err != nil {
		handleErr(err)
		return
	}
	err = runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(time.Second),
	)
	if err != nil {
		handleErr(err)
	}
	return
}

func createReportDir(dir string) (string, error) {
	err := os.MkdirAll(dir, 0o750)
	if err != nil {
		return "", err
	}
	reportFormat := "060102150405"
	name := time.Now().UTC().Format(reportFormat)
	i := 0
	for {
		path := filepath.Join(dir, name)
		if i > 0 {
			path += "_" + strconv.Itoa(i)
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			err := os.Mkdir(path, 0o750)
			if err != nil {
				return "", err
			}
			return path, nil
		}
		i++
	}
}

// SetupStdout bootstraps the OpenTelemetry SDK with the stdout exporters.
func SetupStdout(ctx context.Context, debugDir, version string) (shutdown func(context.Context) error, err error) {
	dir, err := createReportDir(debugDir)
	if err != nil {
		return nil, err
	}
	var shutdownFuncs []func(context.Context) error
	shutdown = func(ctx context.Context) error {
		slog.Info(fmt.Sprintf("Debug reports are saved in %s", dir))
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}
	resource, err := makeResource(ctx, version)
	if err != nil {
		return nil, err
	}
	logFile, err := os.Create(filepath.Join(dir, "logs.json")) //nolint:gosec // Safe file path construction in a controlled directory
	if err != nil {
		return nil, err
	}
	shutdownFuncs = append(shutdownFuncs, func(ctx context.Context) error {
		return logFile.Close()
	})
	logExporter, err := stdoutlog.New(
		stdoutlog.WithPrettyPrint(),
		stdoutlog.WithWriter(logFile),
	)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, logExporter.Shutdown)
	logProvider := log.NewLoggerProvider(
		log.WithProcessor(
			log.NewBatchProcessor(logExporter, log.WithExportMaxBatchSize(1)),
		),
		log.WithResource(resource),
	)
	global.SetLoggerProvider(logProvider)
	traceFile, err := os.Create(filepath.Join(dir, "traces.json")) //nolint:gosec // Safe file path construction in a controlled directory
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, func(ctx context.Context) error {
		return traceFile.Close()
	})
	traceExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(traceFile),
	)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, traceExporter.Shutdown)
	traceProvider := trace.NewTracerProvider(
		trace.WithSyncer(traceExporter),
		trace.WithResource(resource),
	)
	otel.SetTracerProvider(traceProvider)
	metricFile, err := os.Create(filepath.Join(dir, "metrics.json")) //nolint:gosec // Safe file path construction in a controlled directory
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, func(ctx context.Context) error {
		return metricFile.Close()
	})
	metricExporter, err := stdoutmetric.New(
		stdoutmetric.WithPrettyPrint(),
		stdoutmetric.WithWriter(metricFile),
	)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, metricExporter.Shutdown)
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			metric.WithInterval(100*time.Millisecond))),
		metric.WithResource(resource),
	)
	otel.SetMeterProvider(meterProvider)
	err = host.Start()
	if err != nil {
		handleErr(err)
		return
	}
	err = runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(time.Second),
	)
	if err != nil {
		handleErr(err)
	}
	return
}

func makePropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func makeOtelpTracerProvider(ctx context.Context, rs *resource.Resource, otelpURL string) (*trace.TracerProvider, error) {
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(otelpURL),
	)
	if err != nil {
		return nil, err
	}
	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			trace.WithBatchTimeout(time.Second),
		),
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(rs),
	)
	return traceProvider, nil
}

func makeOtelpMeterProvider(ctx context.Context, rs *resource.Resource) (*metric.MeterProvider, error) {
	metricExporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithInsecure())
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			metric.WithInterval(1*time.Second))),
		metric.WithResource(rs),
	)
	return meterProvider, nil
}

func makeOtelpLoggerProvider(ctx context.Context, rs *resource.Resource) (*log.LoggerProvider, error) {
	logExporter, err := otlploghttp.New(ctx, otlploghttp.WithInsecure())
	if err != nil {
		return nil, err
	}
	logProvider := log.NewLoggerProvider(
		log.WithProcessor(
			log.NewBatchProcessor(logExporter),
		),
		log.WithResource(rs),
	)
	return logProvider, nil
}

func makeResource(ctx context.Context, version string) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithOS(),
		resource.WithAttributes(
			attribute.String("service.name", "fabric"),
			attribute.String("version", version),
		),
	)
}
