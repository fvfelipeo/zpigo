package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
	With(key string, value interface{}) Logger
	WithComponent(component string) Logger
}

type ZLogger struct {
	logger zerolog.Logger
}

type Config struct {
	Level     string
	Format    string
	Output    string
	File      string
	Component string
}

func New(config Config) Logger {
	level := parseLevel(config.Level)
	zerolog.SetGlobalLevel(level)

	var output io.Writer
	switch strings.ToLower(config.Output) {
	case "stderr":
		output = os.Stderr
	case "file":
		if config.File != "" {
			file, err := os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				log.Fatal().Err(err).Msg("Erro ao abrir arquivo de log")
			}
			output = file
		} else {
			output = os.Stdout
		}
	default:
		output = os.Stdout
	}

	var logger zerolog.Logger
	switch strings.ToLower(config.Format) {
	case "console":
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
		logger = zerolog.New(output).With().Timestamp().Logger()
	default:
		logger = zerolog.New(output).With().Timestamp().Logger()
	}

	if config.Component != "" {
		logger = logger.With().Str("component", config.Component).Logger()
	}

	return &ZLogger{logger: logger}
}

func NewDefault() Logger {
	return New(Config{
		Level:  "info",
		Format: "console",
		Output: "stdout",
	})
}

func NewForComponent(component string) Logger {
	return New(Config{
		Level:     "info",
		Format:    "console",
		Output:    "stdout",
		Component: component,
	})
}

func (l *ZLogger) Debug(msg string, fields ...interface{}) {
	event := l.logger.Debug()
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *ZLogger) Info(msg string, fields ...interface{}) {
	event := l.logger.Info()
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *ZLogger) Warn(msg string, fields ...interface{}) {
	event := l.logger.Warn()
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *ZLogger) Error(msg string, fields ...interface{}) {
	event := l.logger.Error()
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *ZLogger) Fatal(msg string, fields ...interface{}) {
	event := l.logger.Fatal()
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *ZLogger) With(key string, value interface{}) Logger {
	newLogger := l.logger.With().Interface(key, value).Logger()
	return &ZLogger{logger: newLogger}
}

func (l *ZLogger) WithComponent(component string) Logger {
	newLogger := l.logger.With().Str("component", component).Logger()
	return &ZLogger{logger: newLogger}
}

func (l *ZLogger) addFields(event *zerolog.Event, fields ...interface{}) {
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fmt.Sprintf("%v", fields[i])
			value := fields[i+1]
			event.Interface(key, value)
		}
	}
}

func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

func NewWhatsAppLogger(component string, level string) waLog.Logger {
	return waLog.Stdout(component, level, true)
}

var globalLogger Logger

func Init(config Config) {
	globalLogger = New(config)
}

func InitDefault() {
	globalLogger = NewDefault()
}

func Get() Logger {
	if globalLogger == nil {
		InitDefault()
	}
	return globalLogger
}

func Debug(msg string, fields ...interface{}) {
	Get().Debug(msg, fields...)
}

func Info(msg string, fields ...interface{}) {
	Get().Info(msg, fields...)
}

func Warn(msg string, fields ...interface{}) {
	Get().Warn(msg, fields...)
}

func Error(msg string, fields ...interface{}) {
	Get().Error(msg, fields...)
}

func Fatal(msg string, fields ...interface{}) {
	Get().Fatal(msg, fields...)
}

func With(key string, value interface{}) Logger {
	return Get().With(key, value)
}

func WithComponent(component string) Logger {
	return Get().WithComponent(component)
}
