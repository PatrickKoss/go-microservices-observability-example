package tracing

import (
	"context"
	"encoding/json"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Tracer interface for tracing.
//
//go:generate mockgen -destination=./mock/tracer.go -source=./tracer.go
type Tracer interface {
	// Start a new span.
	Start(ctx context.Context, spanName string) (context.Context, oteltrace.Span)
	StartSpanFromHeader(ctx context.Context, h http.Header, spanName string) (context.Context, oteltrace.Span)
	StartSpanWithLinkToParent(
		ctx context.Context,
		spanName string,
		parentSpanInfo oteltrace.SpanContext,
	) (context.Context, oteltrace.Span)
	StartSpanWithContext(
		ctx context.Context,
		spanName string,
		spanContext oteltrace.SpanContext,
	) (context.Context, oteltrace.Span)
	InjectHTTP(ctx context.Context, h http.Header)
	Shutdown() error
}

// tracer to implement Tracer.
type tracer struct {
	tracer oteltrace.Tracer
	tp     *trace.TracerProvider
}

func (t tracer) StartSpanFromHeader(
	ctx context.Context,
	h http.Header,
	spanName string,
) (context.Context, oteltrace.Span) {
	return t.Start(constructContextFromHeader(ctx, h), spanName)
}

func (t tracer) InjectHTTP(ctx context.Context, h http.Header) {
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(h))
}

func (t tracer) StartSpanWithLinkToParent(
	ctx context.Context,
	spanName string,
	parentSpanInfo oteltrace.SpanContext,
) (context.Context, oteltrace.Span) {
	traceID, err := oteltrace.TraceIDFromHex(parentSpanInfo.TraceID().String())
	if err != nil {
		return ctx, nil
	}

	ctx, err = oteltrace.ContextWithSpanContext(
		ctx,
		oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
			TraceID: traceID,
		}),
	), nil
	if err != nil {
		return ctx, nil
	}

	spanID, err := oteltrace.SpanIDFromHex(parentSpanInfo.SpanID().String())
	if err != nil {
		return ctx, nil
	}

	linkToParentSpan := oteltrace.WithLinks(oteltrace.Link{
		SpanContext: oteltrace.NewSpanContext(
			oteltrace.SpanContextConfig{
				TraceID: traceID,
				SpanID:  spanID,
			},
		),
	})

	return t.tracer.Start(ctx, spanName, linkToParentSpan)
}

func (t tracer) StartSpanWithContext(
	ctx context.Context,
	spanName string,
	spanContext oteltrace.SpanContext,
) (context.Context, oteltrace.Span) {
	receivedSpanContext := spanContext.WithRemote(true)
	ctx = oteltrace.ContextWithRemoteSpanContext(ctx, receivedSpanContext)

	return t.tracer.Start(ctx, spanName)
}

// Start a new span.
func (t tracer) Start(ctx context.Context, spanName string) (context.Context, oteltrace.Span) {
	return t.tracer.Start(ctx, spanName)
}

func (t tracer) Shutdown() error {
	ctx := context.Background()
	_ = t.tp.ForceFlush(ctx)

	return t.tp.Shutdown(ctx)
}

// NewTracer creates a new tracing. And set the service name to appName.
func NewTracer(serviceName string, exporter trace.SpanExporter) Tracer {
	tp := newTraceProvider(serviceName, exporter)

	return tracer{
		tracer: tp.Tracer(serviceName),
		tp:     tp,
	}
}

func constructContextFromHeader(ctx context.Context, h http.Header) context.Context {
	return propagation.TraceContext{}.Extract(ctx, propagation.HeaderCarrier(h))
}

func UnmarshalSpanContext(data []byte) (oteltrace.SpanContext, error) {
	var spanContext transportSpanContext
	err := json.Unmarshal(data, &spanContext)
	if err != nil {
		return oteltrace.SpanContext{}, err
	}

	traceID, err := oteltrace.TraceIDFromHex(spanContext.TraceID)
	if err != nil {
		traceID = oteltrace.TraceID{}
	}

	spanID, err := oteltrace.SpanIDFromHex(spanContext.SpanID)
	if err != nil {
		spanID = oteltrace.SpanID{}
	}

	return oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     spanContext.Remote,
	}), nil
}

// transportSpanContext to unmarshal the SpanContext in a custom unmarshal method. It avoids infinite recursion.
type transportSpanContext struct {
	TraceID    string `json:"TraceID"`
	SpanID     string `json:"SpanID"`
	TraceFlags string `json:"TraceFlags"`
	TraceState string `json:"TraceState"`
	Remote     bool   `json:"Remote"`
}

func newTraceProvider(serviceName string, exporter trace.SpanExporter) *trace.TracerProvider {
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		// Record information about this application in a Resource.
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		)),
	)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{}),
	)

	otel.SetTracerProvider(tp)

	return tp
}

type SpanContext struct {
	SpanContext oteltrace.SpanContext
	TraceID     string `json:"TraceID"`
	SpanID      string `json:"SpanID"`
	TraceFlags  string `json:"TraceFlags"`
	TraceState  string `json:"TraceState"`
	Remote      bool   `json:"Remote"`
}

func NewSpanContext(spanContext oteltrace.SpanContext) SpanContext {
	return SpanContext{
		SpanContext: spanContext,
		TraceID:     spanContext.TraceID().String(),
		SpanID:      spanContext.SpanID().String(),
		TraceFlags:  spanContext.TraceFlags().String(),
		TraceState:  spanContext.TraceState().String(),
		Remote:      spanContext.IsRemote(),
	}
}

func (s *SpanContext) UnmarshalJSON(data []byte) error {
	spanContext, err := UnmarshalSpanContext(data)
	if err != nil {
		return err
	}

	s.SpanContext = spanContext
	s.TraceID = spanContext.TraceID().String()
	s.SpanID = spanContext.SpanID().String()
	s.TraceFlags = spanContext.TraceFlags().String()
	s.TraceState = spanContext.TraceState().String()

	return nil
}

func (s *SpanContext) MarshalJSON() ([]byte, error) {
	return s.SpanContext.MarshalJSON()
}
