package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
)

func TestNewJSONLoggerLevels(t *testing.T) {
	for _, tt := range []struct {
		name  string
		level string
		want  slog.Level
	}{
		{"empty defaults to INFO", "", slog.LevelInfo},
		{"unparseable defaults to INFO", "not-a-level", slog.LevelInfo},
		{"DEBUG", "DEBUG", slog.LevelDebug},
		{"lowercase debug", "debug", slog.LevelDebug},
		{"WARN", "WARN", slog.LevelWarn},
		{"ERROR", "ERROR", slog.LevelError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, levelVar := NewJSONLogger(tt.level)
			if got := levelVar.Level(); got != tt.want {
				t.Errorf("NewJSONLogger(%q) level = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

func TestSetDefaultFromEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "WARN")
	_, levelVar := SetDefaultFromEnv()
	if got := levelVar.Level(); got != slog.LevelWarn {
		t.Errorf("SetDefaultFromEnv() level = %v, want %v", got, slog.LevelWarn)
	}
}

// TestJSONOutput checks that log records really are emitted as JSON with the
// attribute keys we set, which is what the DataDog queries depend on.
func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	levelVar := new(slog.LevelVar)
	levelVar.Set(slog.LevelInfo)
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: levelVar}))

	logger.With(slog.String(KeyTraceID, "trace-123")).
		Info("something happened", slog.Int(KeyOrgID, 38))

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log output is not valid JSON: %v (output: %s)", err, buf.String())
	}
	if record["msg"] != "something happened" {
		t.Errorf("msg = %v, want %q", record["msg"], "something happened")
	}
	if record[KeyTraceID] != "trace-123" {
		t.Errorf("%s = %v, want %q", KeyTraceID, record[KeyTraceID], "trace-123")
	}
	if record[KeyOrgID] != float64(38) {
		t.Errorf("%s = %v, want 38", KeyOrgID, record[KeyOrgID])
	}
}

func TestMain(m *testing.M) {
	// Keep the package's own default-logger side effects out of test output.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	os.Exit(m.Run())
}
