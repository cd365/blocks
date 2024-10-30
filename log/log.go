package log

import (
	"github.com/rs/zerolog"
	"io"
	"os"
)

type Logger struct {
	logger      *zerolog.Logger
	customEvent func(event *zerolog.Event) *zerolog.Event
}

func NewLogger(writer io.Writer) *Logger {
	if writer == nil {
		writer = os.Stdout
	}
	logger := zerolog.New(writer).With().Caller().Timestamp().Logger()
	logger.Level(zerolog.TraceLevel)
	return &Logger{
		logger:      &logger,
		customEvent: func(event *zerolog.Event) *zerolog.Event { return event },
	}
}

func (s *Logger) Level(lvl zerolog.Level) *Logger {
	logger := s.logger.Level(lvl)
	s.logger = &logger
	return s
}

func (s *Logger) GetLevel() zerolog.Level {
	return s.logger.GetLevel()
}

// CustomContext Set common log properties.
func (s *Logger) CustomContext(custom func(ctx zerolog.Context) zerolog.Logger) *Logger {
	if custom != nil {
		ctx := s.logger.With()
		logger := custom(ctx)
		s.logger = &logger
	}
	return s
}

// CustomEvent Set custom properties before calling output log.
func (s *Logger) CustomEvent(customEvent func(event *zerolog.Event) *zerolog.Event) *Logger {
	if customEvent != nil {
		s.customEvent = customEvent
	}
	return s
}

func (s *Logger) GetLogger() *zerolog.Logger {
	return s.logger
}

func (s *Logger) SetLogger(logger *zerolog.Logger) *Logger {
	s.logger = logger
	return s
}

// Output Duplicates the current logger and sets writer as its output.
func (s *Logger) Output(writer io.Writer) *Logger {
	logger := s.logger.Output(writer)
	s.logger = &logger
	return s
}

func (s *Logger) Trace() *zerolog.Event {
	return s.customEvent(s.logger.Trace())
}

func (s *Logger) Debug() *zerolog.Event {
	return s.customEvent(s.logger.Debug())
}

func (s *Logger) Info() *zerolog.Event {
	return s.customEvent(s.logger.Info())
}

func (s *Logger) Warn() *zerolog.Event {
	return s.customEvent(s.logger.Warn())
}

func (s *Logger) Error() *zerolog.Event {
	return s.customEvent(s.logger.Error())
}

func (s *Logger) Fatal() *zerolog.Event {
	return s.customEvent(s.logger.Fatal())
}

func (s *Logger) Panic() *zerolog.Event {
	return s.customEvent(s.logger.Panic())
}

var DefaultLogger = NewLogger(nil)

func Trace() *zerolog.Event {
	return DefaultLogger.Trace()
}

func Debug() *zerolog.Event {
	return DefaultLogger.Debug()
}

func Info() *zerolog.Event {
	return DefaultLogger.Info()
}

func Warn() *zerolog.Event {
	return DefaultLogger.Warn()
}

func Error() *zerolog.Event {
	return DefaultLogger.Error()
}

func Fatal() *zerolog.Event {
	return DefaultLogger.Fatal()
}

func Panic() *zerolog.Event {
	return DefaultLogger.Panic()
}
