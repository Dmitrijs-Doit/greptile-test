package logger

import (
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Logger stores the needed functionality to print a log.
type Logger struct {
	trace    string
	started  time.Time
	severity logging.Severity
	labels   map[string]string
}

func newDefaultLogger() *Logger {
	now := time.Now()
	id, _ := uuid.NewRandom()

	return &Logger{
		started: now,
		trace:   getTrace(now, id.String()),
		labels:  make(map[string]string),
	}
}

// Trace returns the trace stored in logger.
func (l *Logger) Trace() string {
	return l.trace
}

// SetLabel allows to optionally specify key/value labels for log entry.
func (l *Logger) SetLabel(key, value string) {
	l.labels[key] = value
}

// SetLabels allows to optionally add additional labels for log entry.
func (l *Logger) SetLabels(labels map[string]string) {
	for key, value := range labels {
		l.SetLabel(key, value)
	}
}

// End sets the parent logging client with the summarized logging entry.
func (l *Logger) End(ctx *gin.Context) {
	if !cloudLogging {
		return
	}

	e := logging.Entry{
		Trace:    l.trace,
		Severity: l.severity,
		HTTPRequest: &logging.HTTPRequest{
			Request:      ctx.Request,
			Status:       ctx.Writer.Status(),
			Latency:      time.Since(l.started),
			ResponseSize: int64(ctx.Writer.Size()),
		},
		Labels:   l.labels,
		Resource: resource,
	}

	parentLogger.Log(e)
}

func logReqEntry(s logging.Severity, l *Logger, msg string) {
	if s > l.severity {
		l.severity = s
	}

	e := logging.Entry{
		Payload:  msg,
		Severity: s,
		Trace:    l.trace,
		Resource: resource,
	}

	if cloudLogging && childLogger != nil {
		childLogger.Log(e)
	}

	if gin.Mode() != gin.ReleaseMode {
		log.Printf("[%s] %s\n", strings.ToLower(s.String()), msg)
	}
}

func logReq(s logging.Severity, l *Logger, v ...interface{}) {
	logReqEntry(s, l, fmt.Sprint(v...))
}

func (l *Logger) Debug(v ...interface{}) {
	logReq(logging.Debug, l, v...)
}

func (l *Logger) Info(v ...interface{}) {
	logReq(logging.Info, l, v...)
}

func (l *Logger) Print(v ...interface{}) {
	logReq(logging.Info, l, v...)
}

func (l *Logger) Warning(v ...interface{}) {
	logReq(logging.Warning, l, v...)
}

func (l *Logger) Error(v ...interface{}) {
	logReq(logging.Error, l, v...)
}

func (l *Logger) Fatal(v ...interface{}) {
	logReq(logging.Critical, l, v...)
	panic(fmt.Sprint(v...))
}

func logReqf(s logging.Severity, l *Logger, format string, v ...interface{}) {
	logReqEntry(s, l, fmt.Sprintf(format, v...))
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	logReqf(logging.Debug, l, format, v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	logReqf(logging.Info, l, format, v...)
}

func (l *Logger) Printf(format string, v ...interface{}) {
	logReqf(logging.Info, l, format, v...)
}

func (l *Logger) Warningf(format string, v ...interface{}) {
	logReqf(logging.Warning, l, format, v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	logReqf(logging.Error, l, format, v...)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	logReqf(logging.Critical, l, format, v...)
	panic(fmt.Sprintf(format, v...))
}

func logReqln(s logging.Severity, l *Logger, v ...interface{}) {
	logReqEntry(s, l, fmt.Sprintln(v...))
}

func (l *Logger) Debugln(v ...interface{}) {
	logReqln(logging.Debug, l, v...)
}

func (l *Logger) Infoln(v ...interface{}) {
	logReqln(logging.Info, l, v...)
}

func (l *Logger) Println(v ...interface{}) {
	logReqln(logging.Info, l, v...)
}

func (l *Logger) Warningln(v ...interface{}) {
	logReqln(logging.Warning, l, v...)
}

func (l *Logger) Errorln(v ...interface{}) {
	logReqln(logging.Error, l, v...)
}

func (l *Logger) Fatalln(v ...interface{}) {
	logReqln(logging.Critical, l, v...)
	panic(fmt.Sprintln(v...))
}
