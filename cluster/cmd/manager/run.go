package main

import (
	"context"

	"github.com/spf13/viper"

	"cluster/internal/tracing"
)

func run(ctx context.Context) error {
	otelEndponint := viper.GetString("otel-endpoint")
	if otelEndponint == "" {
		return nil
	}
	mainLog.Info("Initializing OpenTelemetry tracer provider", "otel-endpoint", otelEndponint)
	tp, err := tracing.InitTracer(ctx, otelEndponint)
	if err != nil {
		mainLog.Error(err, "Failed to initialize tracer")
		return err
	}
	defer func(ctx context.Context) {
		<-ctx.Done()
		if err := tp.Shutdown(ctx); err != nil {
			mainLog.Error(err, "Failed to shutdown tracer provider")
		}
	}(ctx)
	return nil
}
