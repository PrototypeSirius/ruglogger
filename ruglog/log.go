package logger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelTrace Level = -8
	LevelDebug Level = Level(slog.LevelDebug)
	LevelInfo  Level = Level(slog.LevelInfo)
	LevelWarn  Level = Level(slog.LevelWarn)
	LevelError Level = Level(slog.LevelError)
	LevelFatal Level = 12
)

func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "TRACE"
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
	default:
		return fmt.Sprintf("LEVEL(%d)", int(l))
	}
}

func ParseLevel(value string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "trace":
		return LevelTrace, nil
	case "debug":
		return LevelDebug, nil
	case "info", "":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	case "fatal":
		return LevelFatal, nil
	default:
		return LevelInfo, fmt.Errorf("unknown log level %q", value)
	}
}

type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

type Fields = map[string]any

type Option func(*config) error

type config struct {
	level       Level
	format      Format
	timeFormat  string
	addSource   bool
	outputs     []io.Writer
	closers     []io.Closer
	defaults    Fields
	replaceAttr func([]string, slog.Attr) slog.Attr
	handler     slog.Handler
	exitFunc    func(int)
}

func defaultConfig() config {
	return config{
		level:      LevelInfo,
		format:     FormatJSON,
		timeFormat: time.RFC3339Nano,
		outputs:    []io.Writer{os.Stdout},
		defaults:   Fields{},
		exitFunc:   os.Exit,
	}
}

func WithLevel(level Level) Option {
	return func(cfg *config) error {
		cfg.level = level
		return nil
	}
}

func WithLevelString(value string) Option {
	return func(cfg *config) error {
		level, err := ParseLevel(value)
		if err != nil {
			return err
		}
		cfg.level = level
		return nil
	}
}

func WithFormat(format Format) Option {
	return func(cfg *config) error {
		switch format {
		case FormatJSON, FormatText:
			cfg.format = format
			return nil
		default:
			return fmt.Errorf("unsupported log format %q", format)
		}
	}
}

func WithTimeFormat(format string) Option {
	return func(cfg *config) error {
		if strings.TrimSpace(format) == "" {
			cfg.timeFormat = time.RFC3339Nano
			return nil
		}
		cfg.timeFormat = format
		return nil
	}
}

func WithAddSource(addSource bool) Option {
	return func(cfg *config) error {
		cfg.addSource = addSource
		return nil
	}
}

func WithOutput(output io.Writer) Option {
	return func(cfg *config) error {
		if output == nil {
			return errors.New("logger output cannot be nil")
		}
		cfg.outputs = []io.Writer{output}
		return nil
	}
}

func WithOutputs(outputs ...io.Writer) Option {
	return func(cfg *config) error {
		if len(outputs) == 0 {
			return errors.New("at least one logger output is required")
		}
		cfg.outputs = make([]io.Writer, 0, len(outputs))
		for _, output := range outputs {
			if output == nil {
				return errors.New("logger output cannot be nil")
			}
			cfg.outputs = append(cfg.outputs, output)
		}
		return nil
	}
}

func WithFile(path string) Option {
	return func(cfg *config) error {
		cleanPath := filepath.Clean(strings.TrimSpace(path))
		if cleanPath == "." || cleanPath == "" {
			return errors.New("logger file path cannot be empty")
		}
		file, err := os.OpenFile(cleanPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		cfg.outputs = append(cfg.outputs, file)
		cfg.closers = append(cfg.closers, file)
		return nil
	}
}

func WithFields(fields Fields) Option {
	return func(cfg *config) error {
		cfg.defaults = MergeFields(cfg.defaults, fields)
		return nil
	}
}

func WithField(key string, value any) Option {
	return WithFields(Fields{key: value})
}

func WithReplaceAttr(fn func([]string, slog.Attr) slog.Attr) Option {
	return func(cfg *config) error {
		cfg.replaceAttr = composeReplaceAttr(cfg.replaceAttr, fn)
		return nil
	}
}

func WithHandler(handler slog.Handler) Option {
	return func(cfg *config) error {
		if handler == nil {
			return errors.New("logger handler cannot be nil")
		}
		cfg.handler = handler
		return nil
	}
}

func WithExitFunc(fn func(int)) Option {
	return func(cfg *config) error {
		if fn == nil {
			return errors.New("logger exit function cannot be nil")
		}
		cfg.exitFunc = fn
		return nil
	}
}

type sharedState struct {
	closeOnce sync.Once
	closeErr  error
	closers   []io.Closer
	exitFunc  func(int)
	level     *slog.LevelVar
}

type Logger struct {
	base  *slog.Logger
	state *sharedState
}

func New(opts ...Option) (*Logger, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&cfg); err != nil {
			closeAll(cfg.closers)
			return nil, err
		}
	}

	state := &sharedState{
		closers:  cfg.closers,
		exitFunc: cfg.exitFunc,
		level:    &slog.LevelVar{},
	}
	state.level.Set(slog.Level(cfg.level))

	handler := cfg.handler
	if handler == nil {
		writer := resolveWriter(cfg.outputs)
		options := &slog.HandlerOptions{
			Level:     state.level,
			AddSource: cfg.addSource,
			ReplaceAttr: composeReplaceAttr(
				defaultReplaceAttr(cfg.timeFormat),
				cfg.replaceAttr,
			),
		}

		switch cfg.format {
		case FormatJSON:
			handler = slog.NewJSONHandler(writer, options)
		case FormatText:
			handler = slog.NewTextHandler(writer, options)
		default:
			closeAll(cfg.closers)
			return nil, fmt.Errorf("unsupported log format %q", cfg.format)
		}
	}

	base := slog.New(handler)
	if len(cfg.defaults) > 0 {
		base = base.With(fieldsToArgs(cfg.defaults)...)
	}

	return &Logger{
		base:  base,
		state: state,
	}, nil
}

func MustNew(opts ...Option) *Logger {
	log, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return log
}

func (l *Logger) clone(base *slog.Logger) *Logger {
	return &Logger{
		base:  base,
		state: l.state,
	}
}

func (l *Logger) effective() *Logger {
	if l == nil {
		return Get()
	}
	return l
}

func (l *Logger) Slog() *slog.Logger {
	return l.effective().base
}

func (l *Logger) Close() error {
	if l == nil || l.state == nil {
		return nil
	}
	l.state.closeOnce.Do(func() {
		l.state.closeErr = closeAll(l.state.closers)
	})
	return l.state.closeErr
}

func (l *Logger) SetLevel(level Level) {
	l.effective().state.level.Set(slog.Level(level))
}

func (l *Logger) Level() Level {
	return Level(l.effective().state.level.Level())
}

func (l *Logger) Enabled(ctx context.Context, level Level) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	return l.effective().base.Enabled(ctx, slog.Level(level))
}

func (l *Logger) With(args ...any) *Logger {
	log := l.effective()
	if len(args) == 0 {
		return log
	}
	return log.clone(log.base.With(args...))
}

func (l *Logger) WithFields(fields Fields) *Logger {
	log := l.effective()
	if len(fields) == 0 {
		return log
	}
	return log.clone(log.base.With(fieldsToArgs(fields)...))
}

func (l *Logger) WithField(key string, value any) *Logger {
	return l.WithFields(Fields{key: normalizeFieldValue(value)})
}

func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l.effective()
	}
	return l.WithField("error", err.Error())
}

func (l *Logger) WithAppCode(appCode int) *Logger {
	if appCode == 0 {
		return l.effective()
	}
	return l.WithField("app_code", appCode)
}

func (l *Logger) WithGroup(name string) *Logger {
	log := l.effective()
	if strings.TrimSpace(name) == "" {
		return log
	}
	return log.clone(log.base.WithGroup(name))
}

func (l *Logger) IntoContext(ctx context.Context) context.Context {
	return IntoContext(ctx, l)
}

func (l *Logger) Log(level Level, msg string, err error, appCode int, fields Fields) {
	l.LogContext(context.Background(), level, msg, err, appCode, fields)
}

func (l *Logger) LogContext(ctx context.Context, level Level, msg string, err error, appCode int, fields Fields) {
	log := l.effective()
	if ctx == nil {
		ctx = context.Background()
	}

	merged := MergeFields(fields)
	if err != nil {
		merged["error"] = err.Error()
	}
	if appCode != 0 {
		merged["app_code"] = appCode
	}

	log.base.Log(ctx, slog.Level(level), msg, fieldsToArgs(merged)...)
	if level == LevelFatal {
		log.state.exitFunc(1)
	}
}

func (l *Logger) Trace(msg string, fields Fields) {
	l.Log(LevelTrace, msg, nil, 0, fields)
}

func (l *Logger) TraceContext(ctx context.Context, msg string, fields Fields) {
	l.LogContext(ctx, LevelTrace, msg, nil, 0, fields)
}

func (l *Logger) Debug(msg string, fields Fields) {
	l.Log(LevelDebug, msg, nil, 0, fields)
}

func (l *Logger) DebugContext(ctx context.Context, msg string, fields Fields) {
	l.LogContext(ctx, LevelDebug, msg, nil, 0, fields)
}

func (l *Logger) Info(msg string, fields Fields) {
	l.Log(LevelInfo, msg, nil, 0, fields)
}

func (l *Logger) InfoContext(ctx context.Context, msg string, fields Fields) {
	l.LogContext(ctx, LevelInfo, msg, nil, 0, fields)
}

func (l *Logger) Warn(msg string, appCode int, fields Fields) {
	l.Log(LevelWarn, msg, nil, appCode, fields)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, appCode int, fields Fields) {
	l.LogContext(ctx, LevelWarn, msg, nil, appCode, fields)
}

func (l *Logger) Error(msg string, err error, appCode int, fields Fields) {
	l.Log(LevelError, msg, err, appCode, fields)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, err error, appCode int, fields Fields) {
	l.LogContext(ctx, LevelError, msg, err, appCode, fields)
}

func (l *Logger) Fatal(msg string, err error, appCode int, fields Fields) {
	l.Log(LevelFatal, msg, err, appCode, fields)
}

func (l *Logger) FatalContext(ctx context.Context, msg string, err error, appCode int, fields Fields) {
	l.LogContext(ctx, LevelFatal, msg, err, appCode, fields)
}

func MergeFields(fieldSets ...Fields) Fields {
	merged := Fields{}
	for _, fields := range fieldSets {
		for key, value := range fields {
			merged[key] = normalizeFieldValue(value)
		}
	}
	return merged
}

func fieldsToArgs(fields Fields) []any {
	if len(fields) == 0 {
		return nil
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	args := make([]any, 0, len(keys)*2)
	for _, key := range keys {
		args = append(args, key, normalizeFieldValue(fields[key]))
	}
	return args
}

func normalizeFieldValue(value any) any {
	if err, ok := value.(error); ok && err != nil {
		return err.Error()
	}
	return value
}

func resolveWriter(outputs []io.Writer) io.Writer {
	switch len(outputs) {
	case 0:
		return os.Stdout
	case 1:
		return outputs[0]
	default:
		return io.MultiWriter(outputs...)
	}
}

func defaultReplaceAttr(timeFormat string) func([]string, slog.Attr) slog.Attr {
	return func(_ []string, attr slog.Attr) slog.Attr {
		switch attr.Key {
		case slog.TimeKey:
			attr.Key = "timestamp"
			if attr.Value.Kind() == slog.KindTime {
				attr.Value = slog.StringValue(attr.Value.Time().Format(timeFormat))
			}
		case slog.LevelKey:
			attr.Key = "level"
			attr.Value = slog.StringValue(strings.ToUpper(attr.Value.String()))
		case slog.MessageKey:
			attr.Key = "message"
		}
		return attr
	}
}

func composeReplaceAttr(replacers ...func([]string, slog.Attr) slog.Attr) func([]string, slog.Attr) slog.Attr {
	active := make([]func([]string, slog.Attr) slog.Attr, 0, len(replacers))
	for _, replacer := range replacers {
		if replacer != nil {
			active = append(active, replacer)
		}
	}
	if len(active) == 0 {
		return nil
	}

	return func(groups []string, attr slog.Attr) slog.Attr {
		for _, replacer := range active {
			attr = replacer(groups, attr)
		}
		return attr
	}
}

func closeAll(closers []io.Closer) error {
	var joined error
	for _, closer := range closers {
		if closer == nil {
			continue
		}
		joined = errors.Join(joined, closer.Close())
	}
	return joined
}

type contextKey struct{}

func IntoContext(ctx context.Context, log *Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if log == nil {
		log = Get()
	}
	return context.WithValue(ctx, contextKey{}, log)
}

func FromContext(ctx context.Context) *Logger {
	if ctx == nil {
		return Get()
	}
	log, ok := ctx.Value(contextKey{}).(*Logger)
	if !ok || log == nil {
		return Get()
	}
	return log
}

var (
	defaultMu     sync.RWMutex
	defaultLogger *Logger
)

func Init(level Level, formatTime string, filePath string) error {
	opts := []Option{WithLevel(level)}
	if formatTime != "" {
		opts = append(opts, WithTimeFormat(formatTime))
	}
	if filePath != "" {
		opts = append(opts, WithFile(filePath))
	}
	return InitWithOptions(opts...)
}

func InitWithOptions(opts ...Option) error {
	log, err := New(opts...)
	if err != nil {
		return err
	}

	previous := SetDefault(log)
	if previous != nil {
		_ = previous.Close()
	}
	return nil
}

func MustInit(opts ...Option) *Logger {
	if err := InitWithOptions(opts...); err != nil {
		panic(err)
	}
	return Get()
}

func SetDefault(log *Logger) *Logger {
	defaultMu.Lock()
	defer defaultMu.Unlock()

	previous := defaultLogger
	defaultLogger = log
	return previous
}

func Get() *Logger {
	defaultMu.RLock()
	log := defaultLogger
	defaultMu.RUnlock()
	if log != nil {
		return log
	}

	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultLogger == nil {
		defaultLogger = MustNew()
	}
	return defaultLogger
}

func Close() error {
	defaultMu.Lock()
	log := defaultLogger
	defaultLogger = nil
	defaultMu.Unlock()

	if log == nil {
		return nil
	}
	return log.Close()
}

func ResetDefaultForTest() {
	_ = Close()
}

func Info(msg string, fields Fields) {
	Get().Info(msg, fields)
}

func Trace(msg string, fields Fields) {
	Get().Trace(msg, fields)
}

func Debug(msg string, appCode int, fields Fields) {
	Get().Log(LevelDebug, msg, nil, appCode, fields)
}

func Warn(msg string, appCode int, fields Fields) {
	Get().Warn(msg, appCode, fields)
}

func Error(msg string, err error, appCode int, fields Fields) {
	Get().Error(msg, err, appCode, fields)
}

func Fatal(msg string, err error, appCode int, fields Fields) {
	Get().Fatal(msg, err, appCode, fields)
}
