package logger

import (
	"context"
	"fmt"
	"runtime"

	"cloud.google.com/go/logging"
)

// DetailedLogger stores the needed functionality to print a log.
type DetailedLogger struct {
	Logger
}

// DetailedLoggerFromContext returns the detailed logger that was stored in context.
// If there isn't logger stored, returns a new logger.
func DetailedLoggerFromContext(ctx context.Context) ILogger {
	if d, ok := ctx.Value(CtxDetailedLoggerKey).(*DetailedLogger); ok {
		return d
	}

	return newDetailedLogger()
}

func newDetailedLogger() *DetailedLogger {
	return &DetailedLogger{
		Logger: *newDefaultLogger(),
	}
}

func detailedLogReq(s logging.Severity, l *Logger, v ...interface{}) {
	if _, filename, line, ok := runtime.Caller(2); ok {
		v = append([]interface{}{fmt.Sprintf("%s:%d ", filename, line)}, v...)
	}

	logReqEntry(s, l, fmt.Sprint(v...))
}

func detailedLogReqf(s logging.Severity, l *Logger, format string, v ...interface{}) {
	format = "%s:%d " + format

	if _, filename, line, ok := runtime.Caller(2); ok {
		v = append(append([]interface{}{filename}, line), v...)
	}

	logReqEntry(s, l, fmt.Sprintf(format, v...))
}

func (l *DetailedLogger) Debug(v ...interface{}) {
	detailedLogReq(logging.Debug, &l.Logger, v...)
}

func (l *DetailedLogger) Info(v ...interface{}) {
	detailedLogReq(logging.Info, &l.Logger, v...)
}

func (l *DetailedLogger) Print(v ...interface{}) {
	detailedLogReq(logging.Info, &l.Logger, v...)
}

func (l *DetailedLogger) Warning(v ...interface{}) {
	detailedLogReq(logging.Warning, &l.Logger, v...)
}

func (l *DetailedLogger) Error(v ...interface{}) {
	detailedLogReq(logging.Error, &l.Logger, v...)
}

func (l *DetailedLogger) Fatal(v ...interface{}) {
	detailedLogReq(logging.Critical, &l.Logger, v...)
	panic(fmt.Sprint(v...))
}

func (l *DetailedLogger) Debugf(format string, v ...interface{}) {
	detailedLogReqf(logging.Debug, &l.Logger, format, v...)
}

func (l *DetailedLogger) Infof(format string, v ...interface{}) {
	detailedLogReqf(logging.Info, &l.Logger, format, v...)
}

func (l *DetailedLogger) Printf(format string, v ...interface{}) {
	detailedLogReqf(logging.Info, &l.Logger, format, v...)
}

func (l *DetailedLogger) Warningf(format string, v ...interface{}) {
	detailedLogReqf(logging.Warning, &l.Logger, format, v...)
}

func (l *DetailedLogger) Errorf(format string, v ...interface{}) {
	detailedLogReqf(logging.Error, &l.Logger, format, v...)
}

func (l *DetailedLogger) Fatalf(format string, v ...interface{}) {
	detailedLogReqf(logging.Critical, &l.Logger, format, v...)
	panic(fmt.Sprintf(format, v...))
}
