package logger

import (
	"io"
	"os"
	"sync"

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

func LogOnError(err error, message string, fields ...logrus.Fields) {
	if err == nil {
		return
	}
	entry := Get().WithField("error", err)
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
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	entry.Fatal(message) // .Fatal() = .Error() + os.Exit(1)
}

func ResetForTest() {
	log = nil
	once = sync.Once{}
}
