package handler

import (
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
	"github.com/pennsieve/publishing-service/api/logging"
)

// correlationHeaders are the inbound headers we accept as an existing
// correlation id, in priority order. X-Amzn-Trace-Id is what API Gateway / the
// AWS X-Ray integration supplies; the other two cover callers that set their
// own. Header lookup is case-insensitive because API Gateway v2 lower-cases
// header names but we do not want to depend on that.
var correlationHeaders = []string{"x-amzn-trace-id", "x-request-id", "traceparent"}

// traceID returns the correlation id for this invocation: an inbound
// correlation header if the caller sent one, otherwise a newly generated UUID.
// Nothing is propagated outbound in this pass — the id exists so that every log
// line produced *within* publishing-service while handling one request can be
// found with a single query.
func traceID(request events.APIGatewayV2HTTPRequest) string {
	for _, name := range correlationHeaders {
		for header, value := range request.Headers {
			if value == "" {
				continue
			}
			if equalFold(header, name) {
				return value
			}
		}
	}
	return uuid.NewString()
}

// equalFold is a small ASCII-only case-insensitive compare, avoiding a strings
// import solely for header matching.
func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// newRequestLogger builds the one logger used for the whole invocation. Every
// downstream layer receives this logger (or a child of it via .With) rather
// than reaching for slog.Default, so that the trace id is present on every
// line without any call site having to remember it.
func newRequestLogger(request events.APIGatewayV2HTTPRequest) *slog.Logger {
	return slog.Default().With(
		slog.String(logging.KeyTraceID, traceID(request)),
		slog.String(logging.KeyRequestID, request.RequestContext.RequestID),
	)
}
