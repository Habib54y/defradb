// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package logging

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logger struct {
	name     string
	logger   *zap.Logger
	syncLock sync.RWMutex
}

var _ Logger = (*logger)(nil)

func mustNewLogger(name string) *logger {
	l, err := buildZapLogger(name, Config{})
	if err != nil {
		panic(err)
	}

	return &logger{
		name:   name,
		logger: l,
	}
}

func (l *logger) Debug(ctx context.Context, message string, keyvals ...KV) {
	l.syncLock.RLock()
	defer l.syncLock.RUnlock()

	l.logger.Debug(message, toZapFields(keyvals)...)
}

func (l *logger) Info(ctx context.Context, message string, keyvals ...KV) {
	l.syncLock.RLock()
	defer l.syncLock.RUnlock()

	l.logger.Info(message, toZapFields(keyvals)...)
}

func (l *logger) Warn(ctx context.Context, message string, keyvals ...KV) {
	l.syncLock.RLock()
	defer l.syncLock.RUnlock()

	l.logger.Warn(message, toZapFields(keyvals)...)
}

func (l *logger) Error(ctx context.Context, message string, keyvals ...KV) {
	l.syncLock.RLock()
	defer l.syncLock.RUnlock()

	l.logger.Error(message, toZapFields(keyvals)...)
}

func (l *logger) ErrorE(ctx context.Context, message string, err error, keyvals ...KV) {
	kvs := keyvals
	kvs = append(kvs, NewKV("Error", err))

	l.syncLock.RLock()
	defer l.syncLock.RUnlock()

	l.logger.Error(message, toZapFields(kvs)...)
}

func (l *logger) Fatal(ctx context.Context, message string, keyvals ...KV) {
	l.syncLock.RLock()
	defer l.syncLock.RUnlock()

	l.logger.Fatal(message, toZapFields(keyvals)...)
}

func (l *logger) FatalE(ctx context.Context, message string, err error, keyvals ...KV) {
	kvs := keyvals
	kvs = append(kvs, NewKV("Error", err))

	l.syncLock.RLock()
	defer l.syncLock.RUnlock()

	l.logger.Fatal(message, toZapFields(kvs)...)
}

func (l *logger) Flush() error {
	return l.logger.Sync()
}

func toZapFields(keyvals []KV) []zap.Field {
	result := make([]zap.Field, len(keyvals))
	for i, kv := range keyvals {
		result[i] = zap.Any(kv.key, kv.value)
	}
	return result
}

func (l *logger) ApplyConfig(config Config) {
	newLogger, err := buildZapLogger(l.name, config)
	if err != nil {
		l.logger.Error("Error applying config to logger", zap.Error(err))
		return
	}

	l.syncLock.Lock()
	defer l.syncLock.Unlock()

	// We need sync the old log before swapping it out
	_ = l.logger.Sync()
	l.logger = newLogger
}

func buildZapLogger(name string, config Config) (*zap.Logger, error) {
	defaultConfig := zap.NewProductionConfig()
	defaultConfig.Encoding = "console"
	defaultConfig.EncoderConfig.ConsoleSeparator = ", "
	defaultConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	defaultConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	defaultConfig.DisableStacktrace = true

	if config.Level.HasValue {
		defaultConfig.Level = zap.NewAtomicLevelAt(zapcore.Level(config.Level.LogLevel))
	}

	if config.EnableStackTrace.HasValue {
		defaultConfig.DisableStacktrace = !config.EnableStackTrace.EnableStackTrace
	}

	if config.EncoderFormat.HasValue {
		if config.EncoderFormat.EncoderFormat == JSON {
			defaultConfig.Encoding = "json"
			defaultConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		} else if config.EncoderFormat.EncoderFormat == CSV {
			defaultConfig.Encoding = "console"
		}
	}

	if len(config.OutputPaths) != 0 {
		defaultConfig.OutputPaths = config.OutputPaths[:]
	}

	// We must skip the first caller, as this will always be our wrapper
	newLogger, err := defaultConfig.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}

	return newLogger.Named(name), nil
}