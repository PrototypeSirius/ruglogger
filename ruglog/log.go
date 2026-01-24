package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	}
	return ""
}

type LogEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
	Error     string         `json:"error,omitempty"`
	AppCode   int            `json:"app_code,omitempty"`
}

type Logger struct {
	mu         sync.Mutex
	output     io.Writer
	closer     io.Closer
	level      Level
	formatTime string
}

var (
	instance *Logger
	once     sync.Once
)

func Init(level Level, formatTime string, filePath string) error {
	var err error
	once.Do(func() {
		var out io.Writer = os.Stdout
		var closer io.Closer
		if filePath != "" {
			file, Err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if Err != nil {
				err = Err
				return
			}
			out = file
			closer = file
		}
		if formatTime == "" {
			formatTime = time.RFC3339
		}
		instance = &Logger{
			output:     out,
			closer:     closer,
			level:      level,
			formatTime: formatTime,
		}
	})
	return err
}

func Get() *Logger {
	if instance == nil {
		_ = Init(LevelInfo, "", "")
	}
	return instance
}

func Close() error {
	if instance != nil && instance.closer != nil {
		instance.mu.Lock()
		defer instance.mu.Unlock()
		return instance.closer.Close()
	}
	return nil
}

func (l *Logger) Log(level Level, msg string, err error, appCode int, fields map[string]any) {
	if level < l.level {
		return
	}
	entry := LogEntry{
		Timestamp: l.getTimestamp(),
		Level:     level.String(),
		Message:   msg,
		Fields:    fields,
	}
	if err != nil {
		entry.Error = err.Error()
	}
	if appCode != 0 {
		entry.AppCode = appCode
	}
	data, jErr := json.Marshal(entry)
	if jErr != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal log entry: %v\n", jErr)
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.output.Write(data)
	_, _ = l.output.Write([]byte("\n"))
	if level == LevelFatal {
		os.Exit(1)
	}
}

func (l *Logger) getTimestamp() string {
	return time.Now().Format(l.formatTime)
}

// Info логирует информационное сообщение.
func Info(msg string, fields map[string]any) {
	Get().Log(LevelInfo, msg, nil, 0, fields)
}

// Debug логирует отладочное сообщение.
func Debug(msg string, appCode int, fields map[string]any) {
	Get().Log(LevelDebug, msg, nil, appCode, fields)
}

// Warn логирует предупреждение.
func Warn(msg string, appCode int, fields map[string]any) {
	Get().Log(LevelWarn, msg, nil, appCode, fields)
}

// Error логирует ошибку.
func Error(msg string, err error, appCode int, fields map[string]any) {
	Get().Log(LevelError, msg, err, appCode, fields)
}

// Fatal логирует критическое сообщение.
func Fatal(msg string, err error, appCode int, fields map[string]any) {
	Get().Log(LevelFatal, msg, err, appCode, fields)
}
