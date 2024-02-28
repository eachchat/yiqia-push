package log

import (
	"io"
	"os"

	"github.com/go-kit/log"
)

// NewLogger returns a new logger with the given log level.
func NewLogger(level string) log.Logger {
	var logger log.Logger

	w := log.NewSyncWriter(os.Stdout)
	logger = log.NewLogfmtLogger(w)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.Caller(4))
	switch level {
	case "debug":
		logger = log.With(logger, "level", "debug")
	case "info":
		logger = log.With(logger, "level", "info")
	case "warn":
		logger = log.With(logger, "level", "warn")
	case "error":
		logger = log.With(logger, "level", "error")
	default:
		logger = log.With(logger, "level", "info")
	}
	return &Logger{
		w:      w,
		Logger: logger,
	}
}

type GetWriter interface {
	GetWriter() io.Writer
}

type Logger struct {
	w io.Writer
	log.Logger
}

func (l *Logger) GetWriter() io.Writer {
	return l.w
}
