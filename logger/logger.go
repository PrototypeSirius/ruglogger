package logger

import (
	"errors"
	"io"
	"os"
	"sync"

	"github.com/PrototypeSirius/ruglogger/apperror"
	"github.com/sirupsen/logrus"
)

var (
	log  *logrus.Logger
	once sync.Once
)

type Option func(*logrus.Logger)

func WithOutput(output io.Writer) Option {
	return func(l *logrus.Logger) {
		l.SetOutput(output)
	}
}

func WithLevel(level string) Option {
	return func(l *logrus.Logger) {
		logLvl, err := logrus.ParseLevel(level)
		if err != nil {
			logLvl = logrus.InfoLevel
		}
		l.SetLevel(logLvl)
	}
}

func Init(opts ...Option) {
	once.Do(func() {
		log = logrus.New()

		log.SetOutput(os.Stdout)
		log.SetLevel(logrus.InfoLevel)
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.999Z07:00",
		})

		for _, opt := range opts {
			opt(log)
		}
	})
}

func Get() *logrus.Logger {
	if log == nil {
		panic("The logger has not been initialized, call logger.Init() in main.go")
	}
	return log
}

func SimpleLog(level logrus.Level, message string, fields ...logrus.Fields) {
	var entry logrus.FieldLogger = Get()
	if len(fields) > 0 && fields[0] != nil {
		entry = entry.WithFields(fields[0])
	}

	switch level {
	case logrus.DebugLevel:
		entry.Debug(message)
	case logrus.InfoLevel:
		entry.Info(message)
	case logrus.WarnLevel:
		entry.Warn(message)
	case logrus.ErrorLevel:
		entry.Error(message)
	case logrus.FatalLevel:
		entry.Fatal(message)
	case logrus.PanicLevel:
		entry.Panic(message)
	default:
		entry.Info(message)
	}
}

func Info(message string, fields ...logrus.Fields) {
	SimpleLog(logrus.InfoLevel, message, fields...)
}

func Debug(message string, fields ...logrus.Fields) {
	SimpleLog(logrus.DebugLevel, message, fields...)
}

func Trace(message string, fields ...logrus.Fields) {
	SimpleLog(logrus.TraceLevel, message, fields...)
}

func Warn(message string, fields ...logrus.Fields) {
	SimpleLog(logrus.WarnLevel, message, fields...)
}

func LogOnError(err error, message string, fields ...logrus.Fields) {
	if err == nil {
		return
	}
	entry := Get().WithField("error", err)
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		entry = entry.WithFields(logrus.Fields{"app_code": appErr.AppCode, "message": appErr.Message})
	}

	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	entry.Error(message)
}

func FatalOnError(err error, message string, fields ...logrus.Fields) {
	if err == nil {
		return
	}
	entry := Get().WithField("fatal_error", err)
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		entry = entry.WithFields(logrus.Fields{"app_code": appErr.AppCode, "message": appErr.Message})
	}
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	entry.Fatal(message) // .Fatal() = .Error() + os.Exit(1)
}

func ResetForTest() {
	log = nil
	once = sync.Once{}
}
