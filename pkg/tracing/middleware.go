package tracing

import (
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"net/http"
	"strings"
)

func NewTracingMiddleware(t Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := t.StartSpanFromHeader(r.Context(), r.Header, "middleware")
			defer span.End()

			r = r.WithContext(ctx)
			rw := NewResponseWriter(w)
			next.ServeHTTP(rw, r)

			// Append meta data to span
			span.SetAttributes(attribute.String("http.method", strings.ToUpper(r.Method)))
			span.SetAttributes(attribute.String("http.url", r.URL.String()))
			span.SetAttributes(
				attribute.String("http.status_code", fmt.Sprintf("%d", rw.Status())),
			)

			t.InjectHTTP(ctx, w.Header())
		})
	}
}

// NewResponseWriter creates a new ResponseWriter from a http.ResponseWriter.
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

// ResponseWriter is a wrapper around http.ResponseWriter that provides extra information about
// the response. It is recommended that middleware handlers use this construct to wrap a responsewriter
// if the status of the response is used in the middleware.
type ResponseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader saves the status code and calls the original ResponseWriter's WriteHeader.
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Status returns the status code of the response or 200 if the response has not been
// written (as this is the HTTP default).
func (rw *ResponseWriter) Status() int {
	return rw.status
}
