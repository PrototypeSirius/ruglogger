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

// Option определяет тип для функциональных опций, используемых при конфигурации логгера.
// Этот паттерн позволяет гибко настраивать логгер, передавая различные функции-конфигураторы.
type Option func(*logrus.Logger)

// WithOutput устанавливает место назначения для вывода логов (например, os.Stdout или файл).
// По умолчанию используется os.Stdout.
func WithOutput(output io.Writer) Option {
	return func(l *logrus.Logger) {
		l.SetOutput(output)
	}
}

// WithLevel устанавливает минимальный уровень логирования (например, "debug", "info", "warn", "error").
// Если передан некорректный уровень, по умолчанию будет использоваться "info".
func WithLevel(level string) Option {
	return func(l *logrus.Logger) {
		logLvl, err := logrus.ParseLevel(level)
		if err != nil {
			logLvl = logrus.InfoLevel
		}
		l.SetLevel(logLvl)
	}
}

// Init инициализирует синглтон-логгер с заданными опциями.
// Эту функцию необходимо вызвать один раз в самом начале работы приложения, как правило, в main.go.
// Все последующие вызовы Init будут проигнорированы.
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

// Get возвращает настроенный экземпляр логгера.
// Если логгер не был инициализирован через Init(), функция вызовет панику.
// Это сделано намеренно (fail-fast), чтобы сразу обнаружить ошибку в конфигурации приложения,
// а не получать nil pointer exception в случайных местах.
func Get() *logrus.Logger {
	if log == nil {
		panic("The logger has not been initialized, call logger.Init() in main.go")
	}
	return log
}
