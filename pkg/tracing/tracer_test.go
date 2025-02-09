package tracing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
	otelTrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

const dummyURL = "https://api.stackit.cloud"

var lock = sync.Mutex{}

var (
	errGeneric   = errors.New("something went wrong")
	urlParsed, _ = url.Parse(dummyURL)
)

func TestTracer_BasicFlow(t *testing.T) {
	t.Parallel()

	lock.Lock()
	defer lock.Unlock()

	originalStdout := os.Stdout
	defer func() {
		os.Stdout = originalStdout
	}()

	r, w := getReaderWriterFile(t)
	exporter := newConsoleExporter(t, w)
	testTracer := NewTracer("test-server", exporter)

	req := &http.Request{
		URL:    urlParsed,
		Header: map[string][]string{},
	}
	_, s := testTracer.Start(req.Context(), "test")

	s.SetAttributes(attribute.String("test", "test"))
	s.End()

	shutdownTracer(t, testTracer)

	closeWriter(t, w)
	out := readOutput(t, r)

	if !strings.Contains(string(out), "Status") {
		t.Errorf("No status found")
	}
}

func TestTracer_TraceThroughHeader(t *testing.T) {
	t.Parallel()

	lock.Lock()
	defer lock.Unlock()

	originalStdout := os.Stdout
	defer func() {
		os.Stdout = originalStdout
	}()

	r, w := getReaderWriterFile(t)
	exporter := newConsoleExporter(t, w)
	testTracer := NewTracer("test-server", exporter)

	reqA := &http.Request{
		URL:    urlParsed,
		Header: http.Header{},
	}

	ctx, s := testTracer.Start(reqA.Context(), "test init")

	// {version}-{trace_id}-{span_id}-{trace_flags}
	testTracer.InjectHTTP(ctx, reqA.Header)

	_, s1 := testTracer.StartSpanFromHeader(reqA.Context(), reqA.Header, "test A")
	_, s2 := testTracer.StartSpanFromHeader(reqA.Context(), reqA.Header, "test B")

	s1.End()
	s2.End()
	s.End()

	shutdownTracer(t, testTracer)

	closeWriter(t, w)
	out := readOutput(t, r)

	replaced := strings.ReplaceAll(string(out), "}\n{", "},\n{")
	result := make([]spanJSON, 3)
	err := json.Unmarshal([]byte(fmt.Sprintf("[%s]", replaced)), &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal json: %v", err)
	}

	traceIDInit, traceIDA, traceIDB, parentTraceIDA, parentTraceIDB := getTestTracesThroughHeader(
		result,
	)

	assertTracesThroughHeader(t, s, traceIDInit, traceIDA, parentTraceIDA, traceIDB, parentTraceIDB)
}

func TestTracer_TraceExternalHTTPRequest(t *testing.T) {
	t.Parallel()

	tests := getExternalHTTPRequestTests()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			traceMock := &mockTracer{}

			_, s := traceMock.Start(tt.args.req.Context(), "test")
			keyValues := []attribute.KeyValue{
				attribute.String("request.URL", tt.args.req.URL.String()),
				attribute.String("request.method", tt.args.req.Method),
				attribute.Int("request.status", tt.args.status),
				attribute.String("request.time", tt.args.requestDuration.String()),
			}
			s.SetAttributes(keyValues...)
			s.RecordError(tt.args.err)
			s.End()

			resultedSpan := traceMock.mockSpan

			if tt.expectedSpan.endCalled != resultedSpan.endCalled {
				t.Errorf("Span should be ended")

				return
			}
			if !errors.Is(tt.expectedSpan.errorRecorded, resultedSpan.errorRecorded) {
				t.Errorf(
					"Expected error %s, got %s",
					tt.expectedSpan.errorRecorded,
					resultedSpan.errorRecorded,
				)

				return
			}
			if valid, missing := validateAttributes(tt.expectedSpan.attributes, resultedSpan.attributes); !valid {
				t.Errorf("Missing attribute %s", missing.Key)

				return
			}
		})
	}
}

func TestTracer_StartSpanWithLinkToParent(t *testing.T) {
	t.Parallel()

	lock.Lock()
	defer lock.Unlock()

	originalStdout := os.Stdout
	defer func() {
		os.Stdout = originalStdout
	}()

	r, w := getReaderWriterFile(t)
	exporter := newConsoleExporter(t, w)
	testTracer := NewTracer("test-server", exporter)

	traceID, err := otelTrace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("Failed to create traceID: %v", err)
	}

	spanID, err := otelTrace.SpanIDFromHex("0123456789abcdef")
	if err != nil {
		t.Fatalf("Failed to create spanID: %v", err)
	}

	parentSpanInfo := otelTrace.NewSpanContext(otelTrace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	})

	ctx := context.Background()
	_, s := testTracer.StartSpanWithLinkToParent(ctx, "testSpan", parentSpanInfo)
	s.End()

	shutdownTracer(t, testTracer)

	closeWriter(t, w)
	out := readOutput(t, r)

	if !strings.Contains(string(out), "Status") {
		t.Errorf("No status found")
	}
	if !strings.Contains(string(out), traceID.String()) {
		t.Errorf("No traceID found")
	}
	if !strings.Contains(string(out), spanID.String()) {
		t.Errorf("No spanID found")
	}
}

func TestTracer_StartSpanWithContext(t *testing.T) {
	t.Parallel()

	lock.Lock()
	defer lock.Unlock()

	originalStdout := os.Stdout
	defer func() {
		os.Stdout = originalStdout
	}()

	r, w := getReaderWriterFile(t)
	exporter := newConsoleExporter(t, w)
	testTracer := NewTracer("test-server", exporter)

	_, s := testTracer.Start(context.Background(), "testSpan")
	s.End()

	// we want to marshal the trace and then send it somewhere else
	b, _ := json.Marshal(s.SpanContext())
	parentSpanInfo, err := UnmarshalSpanContext(b)
	if err != nil {
		t.Fatalf("Failed to unmarshal span context: %v", err)
	}

	_, s2 := testTracer.StartSpanWithContext(context.Background(), "testSpan2", parentSpanInfo)
	s2.End()

	shutdownTracer(t, testTracer)

	closeWriter(t, w)
	out := readOutput(t, r)
	stringOut := string(out)

	if !strings.Contains(stringOut, "Status") {
		t.Errorf("No status found")
	}
	if !strings.Contains(stringOut, s.SpanContext().TraceID().String()) {
		t.Errorf("No traceID found")
	}
	if strings.Count(stringOut, "00000000000000000000000000000000") > 1 {
		t.Errorf("New trace ID found")
	}
}

type testExternalHTTPRequestArgs struct {
	req             *http.Request
	status          int
	requestDuration time.Duration
	err             error
}

func getExternalHTTPRequestTests() []struct {
	name         string
	args         testExternalHTTPRequestArgs
	expectedSpan *mockSpan
} {
	tests := []struct {
		name         string
		args         testExternalHTTPRequestArgs
		expectedSpan *mockSpan
	}{
		{
			name: "HTTP Request creates span with correct attributes",
			args: testExternalHTTPRequestArgs{
				req: &http.Request{
					Method: http.MethodPost,
					URL:    urlParsed,
					Response: &http.Response{
						StatusCode: http.StatusAccepted,
					},
					Header: map[string][]string{},
				},
				status:          http.StatusAccepted,
				requestDuration: time.Second * 10,
				err:             nil,
			},
			expectedSpan: &mockSpan{
				endCalled:     true,
				errorRecorded: nil,
				attributes: []attribute.KeyValue{
					attribute.String("request.URL", dummyURL),
					attribute.String("request.method", http.MethodPost),
					attribute.Int("request.status", http.StatusAccepted),
					attribute.String("request.time", "10s"),
				},
			},
		},

		{
			name: "Failing HTTP Request creates span error status",
			args: testExternalHTTPRequestArgs{
				req: &http.Request{
					Method: http.MethodPost,
					URL:    urlParsed,
					Response: &http.Response{
						StatusCode: http.StatusAccepted,
					},
					Header: map[string][]string{},
				},
				status:          http.StatusInternalServerError,
				requestDuration: time.Second * 2,
				err:             errGeneric,
			},
			expectedSpan: &mockSpan{
				endCalled:     true,
				errorRecorded: errGeneric,
				attributes: []attribute.KeyValue{
					attribute.String("request.URL", dummyURL),
					attribute.String("request.method", http.MethodPost),
					attribute.Int("request.status", http.StatusInternalServerError),
					attribute.String("request.time", "2s"),
				},
			},
		},
	}

	return tests
}

func validateAttributes(got []attribute.KeyValue, expected []attribute.KeyValue) (bool, *attribute.KeyValue) {
	for _, gotAttribute := range got {
		if !compareAttributes(gotAttribute, expected) {
			return false, &gotAttribute
		}
	}

	return true, nil
}

func compareAttributes(gotAttribute attribute.KeyValue, expected []attribute.KeyValue) bool {
	for _, expectedAttribute := range expected {
		if gotAttribute.Key == expectedAttribute.Key && gotAttribute.Value == expectedAttribute.Value {
			return true
		}
	}

	return false
}

func assertTracesThroughHeader(
	t *testing.T,
	s otelTrace.Span,
	traceIDInit string,
	traceIDA string,
	parentTraceIDA string,
	traceIDB string,
	parentTraceIDB string,
) {
	t.Helper()

	if traceIDInit != s.SpanContext().TraceID().String() {
		t.Errorf("TraceID mismatch: %s != %s", traceIDInit, s.SpanContext().TraceID())
	}

	if traceIDA != s.SpanContext().TraceID().String() {
		t.Errorf("TraceID mismatch: %s != %s", traceIDA, s.SpanContext().TraceID())
	}

	if traceIDB != s.SpanContext().TraceID().String() {
		t.Errorf("TraceID mismatch: %s != %s", traceIDB, s.SpanContext().TraceID())
	}

	if parentTraceIDA != s.SpanContext().TraceID().String() {
		t.Errorf("Parent TraceID mismatch: %s != %s", parentTraceIDA, s.SpanContext().TraceID())
	}

	if parentTraceIDB != s.SpanContext().TraceID().String() {
		t.Errorf("Parent TraceID mismatch: %s != %s", parentTraceIDB, s.SpanContext().TraceID())
	}
}

func getTestTracesThroughHeader(result []spanJSON) (string, string, string, string, string) {
	var traceIDInit, traceIDA, traceIDB, parentTraceIDA, parentTraceIDB string

	for _, res := range result {
		switch res.Name {
		case "test init":
			traceIDInit = res.SpanContext.TraceID
		case "test A":
			traceIDA = res.SpanContext.TraceID
			parentTraceIDA = res.Parent.TraceID
		case "test B":
			traceIDB = res.SpanContext.TraceID
			parentTraceIDB = res.Parent.TraceID
		default:
			continue
		}
	}

	return traceIDInit, traceIDA, traceIDB, parentTraceIDA, parentTraceIDB
}

func readOutput(t *testing.T, r *os.File) []byte {
	t.Helper()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Errorf("Failed to read output: %v", err)
	}
	if len(out) == 0 {
		t.Errorf("No output found")
	}
	err = r.Close()
	if err != nil {
		t.Errorf("Failed to close reader: %v", err)
	}

	return out
}

func closeWriter(t *testing.T, w *os.File) {
	t.Helper()

	err := w.Close()
	if err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
}

func shutdownTracer(t *testing.T, testTracer Tracer) {
	t.Helper()

	err := testTracer.Shutdown()
	if err != nil {
		t.Fatalf("failed to shutdown tracing: %v", err)
	}
}

func getReaderWriterFile(t *testing.T) (*os.File, *os.File) {
	t.Helper()

	r, w, err := os.Pipe()
	os.Stdout = w

	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	return r, w
}

type spanJSON struct {
	Name        string `json:"Name"`
	SpanContext struct {
		TraceID string `json:"TraceID"`
	} `json:"SpanContext"`
	Parent struct {
		TraceID string `json:"TraceID"`
	} `json:"Parent"`
}

// newConsoleExporter returns a console exporter for local setup.
func newConsoleExporter(t *testing.T, w io.Writer) trace.SpanExporter {
	t.Helper()

	exporter, err := stdouttrace.New(
		stdouttrace.WithWriter(w),
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		t.Fatalf("failed to create console exporter: %v", exporter)
	}

	return exporter
}

type mockTracer struct {
	ctx      context.Context
	name     string
	opts     []otelTrace.SpanStartOption
	mockSpan mockSpan
}

func (m *mockTracer) Start(
	ctx context.Context,
	name string,
	opts ...otelTrace.SpanStartOption,
) (context.Context, otelTrace.Span) {
	m.ctx = ctx
	m.name = name
	m.opts = opts
	m.mockSpan = mockSpan{}

	return ctx, &m.mockSpan
}

type mockSpan struct {
	embedded.Span
	endCalled     bool
	errorRecorded error
	attributes    []attribute.KeyValue
}

func (m *mockSpan) SetAttributes(kv ...attribute.KeyValue) {
	m.attributes = append(m.attributes, kv...)
}

func (m *mockSpan) End(_ ...otelTrace.SpanEndOption) {
	m.endCalled = true
}

func (m *mockSpan) AddEvent(_ string, _ ...otelTrace.EventOption) {
	panic("implement me")
}

func (m *mockSpan) IsRecording() bool {
	panic("implement me")
}

func (m *mockSpan) RecordError(err error, _ ...otelTrace.EventOption) {
	m.errorRecorded = err
}

func (m *mockSpan) SpanContext() otelTrace.SpanContext {
	return otelTrace.NewSpanContext(otelTrace.SpanContextConfig{TraceID: [16]byte{1}})
}

func (m *mockSpan) SetStatus(_ codes.Code, _ string) {
	panic("implement me")
}

func (m *mockSpan) SetName(_ string) {
	panic("implement me")
}

func (m *mockSpan) TracerProvider() otelTrace.TracerProvider {
	panic("implement me")
}

func (m *mockSpan) AddLink(link otelTrace.Link) {
	panic("implement me")
}
