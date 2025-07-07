package internal

import (
	"log/slog"

	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

func AppInit() {
	initLog()
}

func initLog() {
	zapL := zap.Must(zap.NewProduction())
	defer zapL.Sync()

	zaphandler := zapslog.NewHandler(zapL.Core(), zapslog.WithCaller(true), zapslog.WithCallerSkip(1), zapslog.AddStacktraceAt(slog.LevelError))
	logger := slog.New(zaphandler)

	slog.SetDefault(logger)

	slog.Debug("this is a debug log")
	slog.Info("this is a info log")
	slog.Warn("this is a warn log")
	slog.Error("this is a error log")
}
