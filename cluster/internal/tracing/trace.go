// Package tracing 提供了 OpenTelemetry 的初始化和注入功能
package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func InitTracer(ctx context.Context, endpoint string) (*sdktrace.TracerProvider, error) {
	// 创建 Jaeger 导出器
	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpointURL(endpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	// 配置 TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("cloudterm-cluster-operator"),
		)),
		sdktrace.WithSampler(sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(0.1), // 生产环境建议降低采样率
		)),
	)

	// 设置全局 TracerProvider 和传播器
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return tp, nil
}

func InjectTraceContext(ctx context.Context) map[string]string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier
}

func LogWithTrace(ctx context.Context) context.Context {
	sc := trace.SpanContextFromContext(ctx)
	if sc.HasTraceID() {
		log := logf.FromContext(ctx).WithValues("trace_id", sc.TraceID().String())
		ctx = logf.IntoContext(ctx, log)
	}
	return ctx
}
