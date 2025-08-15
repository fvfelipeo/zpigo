package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Logger interface {
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
	Fatal(msg string, fields ...any)
	With(key string, value any) Logger
	WithComponent(component string) Logger
}

type Config struct {
	Level     string
	Format    string
	Output    string
	File      string
	Component string
}

type ZLogger struct {
	logger zerolog.Logger
}

func New(config Config) Logger {
	level := parseLevel(config.Level)
	zerolog.SetGlobalLevel(level)

	output := createOutput(config)
	logger := createLogger(config, output)

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

func (l *ZLogger) Debug(msg string, fields ...any) {
	event := l.logger.Debug()
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *ZLogger) Info(msg string, fields ...any) {
	event := l.logger.Info()
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *ZLogger) Warn(msg string, fields ...any) {
	event := l.logger.Warn()
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *ZLogger) Error(msg string, fields ...any) {
	event := l.logger.Error()
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *ZLogger) Fatal(msg string, fields ...any) {
	event := l.logger.Fatal()
	l.addFields(event, fields...)
	event.Msg(msg)
}

func (l *ZLogger) With(key string, value any) Logger {
	newLogger := l.logger.With().Interface(key, value).Logger()
	return &ZLogger{logger: newLogger}
}

func (l *ZLogger) WithComponent(component string) Logger {
	newLogger := l.logger.With().Str("component", component).Logger()
	return &ZLogger{logger: newLogger}
}

func (l *ZLogger) addFields(event *zerolog.Event, fields ...any) {
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fmt.Sprintf("%v", fields[i])
			value := fields[i+1]
			event.Interface(key, value)
		}
	}
}

func createOutput(config Config) io.Writer {
	switch strings.ToLower(config.Output) {
	case "stderr":
		return os.Stderr
	case "file":
		if config.File != "" {
			file, err := os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				log.Fatalf("Erro ao abrir arquivo de log: %v", err)
				return os.Stdout
			}
			return file
		}
		return os.Stdout
	default:
		return os.Stdout
	}
}

func createLogger(config Config, output io.Writer) zerolog.Logger {
	switch strings.ToLower(config.Format) {
	case "console":
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
		return zerolog.New(output).With().Timestamp().Logger()
	default:
		return zerolog.New(output).With().Timestamp().Logger()
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

type WAAdapter struct {
	logger Logger
}

func (w *WAAdapter) Debugf(msg string, args ...any) {
	w.logger.Debug(fmt.Sprintf(msg, args...))
}

func (w *WAAdapter) Infof(msg string, args ...any) {
	w.logger.Info(fmt.Sprintf(msg, args...))
}

func (w *WAAdapter) Warnf(msg string, args ...any) {
	w.logger.Warn(fmt.Sprintf(msg, args...))
}

func (w *WAAdapter) Errorf(msg string, args ...any) {
	w.logger.Error(fmt.Sprintf(msg, args...))
}

func (w *WAAdapter) Sub(module string) waLog.Logger {
	return &WAAdapter{
		logger: w.logger.WithComponent(module),
	}
}

func ForWhatsApp(component string) waLog.Logger {
	return &WAAdapter{logger: NewForComponent(component)}
}

func NewWhatsAppLogger(component string, level string) waLog.Logger {
	return ForWhatsApp(component)
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

func Debug(msg string, fields ...any) {
	Get().Debug(msg, fields...)
}

func Info(msg string, fields ...any) {
	Get().Info(msg, fields...)
}

func Warn(msg string, fields ...any) {
	Get().Warn(msg, fields...)
}

func Error(msg string, fields ...any) {
	Get().Error(msg, fields...)
}

func Fatal(msg string, fields ...any) {
	Get().Fatal(msg, fields...)
}

func With(key string, value any) Logger {
	return Get().With(key, value)
}

func WithComponent(component string) Logger {
	return Get().WithComponent(component)
}
