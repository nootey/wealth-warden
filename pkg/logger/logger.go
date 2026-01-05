package logging

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitLogger(release bool) *zap.Logger {

	var cfg zap.Config

	if release {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	// Build log file path
	logFile := getLogFilePath()

	cfg.OutputPaths = []string{
		"stdout", // keep console output
		logFile,
	}
	cfg.ErrorOutputPaths = []string{
		"stderr",
		logFile,
	}

	// Set log level
	cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)

	logger, err := cfg.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build logger: %v", err))
	}

	return logger
}

func getLogFilePath() string {
	const logDir = "logs"

	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create log directory: %v", err))
	}

	return filepath.Join(logDir, "app.log")
}
