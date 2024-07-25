package logger

import "github.com/gin-gonic/gin"

//go:generate mockery --name ILogger --output ./mocks
type ILogger interface {
	Trace() string
	SetLabel(key, value string)
	SetLabels(labels map[string]string)
	End(ctx *gin.Context)
	Debug(v ...interface{})
	Info(v ...interface{})
	Print(v ...interface{})
	Warning(v ...interface{})
	Error(v ...interface{})
	Fatal(v ...interface{})
	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Printf(format string, v ...interface{})
	Warningf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Fatalf(format string, v ...interface{})
	Debugln(v ...interface{})
	Infoln(v ...interface{})
	Println(v ...interface{})
	Warningln(v ...interface{})
	Errorln(v ...interface{})
	Fatalln(v ...interface{})
}
