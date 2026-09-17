package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/publishing-service/api/logging"
)

func TestTraceIDUsesInboundHeader(t *testing.T) {
	for _, tt := range []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{
			name:    "X-Amzn-Trace-Id",
			headers: map[string]string{"X-Amzn-Trace-Id": "Root=1-5759e988-abc"},
			want:    "Root=1-5759e988-abc",
		},
		{
			name:    "lower-cased as API Gateway v2 sends it",
			headers: map[string]string{"x-amzn-trace-id": "Root=1-5759e988-def"},
			want:    "Root=1-5759e988-def",
		},
		{
			name:    "X-Request-Id",
			headers: map[string]string{"X-Request-Id": "req-123"},
			want:    "req-123",
		},
		{
			name:    "traceparent",
			headers: map[string]string{"traceparent": "00-abc-def-01"},
			want:    "00-abc-def-01",
		},
		{
			name:    "X-Amzn-Trace-Id wins over the others",
			headers: map[string]string{"x-request-id": "req-123", "x-amzn-trace-id": "amzn-wins"},
			want:    "amzn-wins",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := traceID(events.APIGatewayV2HTTPRequest{Headers: tt.headers})
			if got != tt.want {
				t.Errorf("traceID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTraceIDGeneratesWhenAbsent(t *testing.T) {
	for _, headers := range []map[string]string{
		nil,
		{},
		{"content-type": "application/json"},
		{"x-amzn-trace-id": ""}, // present but empty must not be used
	} {
		got := traceID(events.APIGatewayV2HTTPRequest{Headers: headers})
		if got == "" {
			t.Fatalf("traceID() returned empty for headers %v; want a generated id", headers)
		}
		if len(got) != 36 {
			t.Errorf("traceID() = %q (len %d), want a 36-char UUID", got, len(got))
		}
	}
}

func TestTraceIDGeneratesDistinctIDs(t *testing.T) {
	request := events.APIGatewayV2HTTPRequest{}
	if a, b := traceID(request), traceID(request); a == b {
		t.Errorf("traceID() returned the same generated id twice: %q", a)
	}
}

func TestEqualFold(t *testing.T) {
	for _, tt := range []struct {
		a, b string
		want bool
	}{
		{"x-request-id", "X-Request-Id", true},
		{"X-AMZN-TRACE-ID", "x-amzn-trace-id", true},
		{"traceparent", "traceparent", true},
		{"x-request-id", "x-request-idx", false},
		{"x-request-id", "x-requesu-id", false},
		{"", "", true},
	} {
		if got := equalFold(tt.a, tt.b); got != tt.want {
			t.Errorf("equalFold(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

// TestNewRequestLoggerCarriesTraceID checks that every line written through the
// request-scoped logger really carries the trace id and the API Gateway request
// id — the whole point of the correlation work.
func TestNewRequestLoggerCarriesTraceID(t *testing.T) {
	var buf bytes.Buffer
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(original) })

	request := events.APIGatewayV2HTTPRequest{
		Headers: map[string]string{"x-amzn-trace-id": "trace-abc"},
	}
	request.RequestContext.RequestID = "apigw-req-1"

	newRequestLogger(request).Info("handling request")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log output is not valid JSON: %v (output: %s)", err, buf.String())
	}
	if record[logging.KeyTraceID] != "trace-abc" {
		t.Errorf("%s = %v, want %q", logging.KeyTraceID, record[logging.KeyTraceID], "trace-abc")
	}
	if record[logging.KeyRequestID] != "apigw-req-1" {
		t.Errorf("%s = %v, want %q", logging.KeyRequestID, record[logging.KeyRequestID], "apigw-req-1")
	}
}
