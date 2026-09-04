// Package logging configures the service's structured logger (log/slog): a
// JSON handler writing to stdout at a level taken from LOG_LEVEL, with a
// testable pure constructor (NewJSONLogger) separated from the side-effecting
// SetDefaultLogger. Replaces sirupsen/logrus.
//
// This package lives in the api module so that both the api module and the
// lambda/service module (which depends on api) share one logger definition.
package logging

import (
	"log/slog"
	"os"
)

// SetDefaultLogger builds a JSON logger at the given level and installs it as
// slog.Default. Returns the logger and its LevelVar (so the level can be
// changed at runtime). An unparseable level falls back to INFO.
func SetDefaultLogger(level string) (*slog.Logger, *slog.LevelVar) {
	logger, levelVar := NewJSONLogger(level)
	slog.SetDefault(logger)
	slog.Debug("log level set", slog.String("level", levelVar.String()))
	return logger, levelVar
}

// SetDefaultFromEnv installs the default logger using LOG_LEVEL from the
// environment (INFO when unset or invalid). Convenience wrapper for the Lambda
// entrypoint, whose deployment already sets LOG_LEVEL.
func SetDefaultFromEnv() (*slog.Logger, *slog.LevelVar) {
	return SetDefaultLogger(os.Getenv("LOG_LEVEL"))
}

// NewJSONLogger returns a JSON-handler slog.Logger at the given level (INFO if
// the string is empty or unparseable) plus its LevelVar. Pure constructor: no
// global side effects, so tests can build isolated loggers.
func NewJSONLogger(level string) (*slog.Logger, *slog.LevelVar) {
	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(level)); err != nil {
		if level != "" {
			slog.Error("error unmarshalling log level value",
				slog.String("logLevel", level),
				slog.Any("error", err))
		}
		logLevel = slog.LevelInfo
	}

	levelVar := new(slog.LevelVar)
	levelVar.Set(logLevel)

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: levelVar})
	return slog.New(handler), levelVar
}
