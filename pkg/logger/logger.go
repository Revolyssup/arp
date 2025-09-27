package logger

import (
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

// TODO: Add structured logging with fields
type Logger struct {
	*log.Logger
	componentChain string // This will store the full component hierarchy
}

type Level = log.Level

const (
	LevelDebug = log.DebugLevel
	LevelInfo  = log.InfoLevel
	LevelWarn  = log.WarnLevel
	LevelError = log.ErrorLevel
)

func New(level Level) *Logger {
	logger := log.New(os.Stderr)
	logger.SetLevel(level)
	logger.SetTimeFormat("2006-01-02 15:04:05")
	logger.SetReportCaller(false)

	return &Logger{
		Logger: logger,
		// Root logger has empty component chain
	}
}

func (l *Logger) WithComponent(component string) *Logger {
	// Build the component chain
	var newChain string
	if l.componentChain == "" {
		newChain = component
	} else {
		newChain = l.componentChain + "->" + component
	}

	// Create a new logger with the updated component chain
	newLogger := log.New(os.Stderr)
	newLogger.SetLevel(l.GetLevel())
	newLogger.SetTimeFormat("2006-01-02 15:04:05")
	newLogger.SetReportCaller(false)

	// Set the prefix to show the full component hierarchy
	newLogger.SetPrefix("[" + newChain + "]")

	return &Logger{
		Logger:         newLogger,
		componentChain: newChain,
	}
}

func (l *Logger) GetLevel() Level {
	return l.Logger.GetLevel()
}

func (l *Logger) SetLevel(level Level) {
	l.Logger.SetLevel(level)
}

func SetLogLevel(levelStr string) Level {
	level, err := log.ParseLevel(strings.ToLower(levelStr))
	if err != nil {
		return LevelInfo
	}
	return level
}
