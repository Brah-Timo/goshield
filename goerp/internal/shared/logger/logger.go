package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	level  Level
	logger *log.Logger
}

var Default = New(INFO)

func New(level Level) *Logger {
	return &Logger{
		level:  level,
		logger: log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) log(level Level, prefix, msg string, args ...interface{}) {
	if level < l.level {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formatted := fmt.Sprintf(msg, args...)
	l.logger.Printf("[%s] %s %s\n", timestamp, prefix, formatted)
}

func (l *Logger) Debug(msg string, args ...interface{}) { l.log(DEBUG, "DEBUG", msg, args...) }
func (l *Logger) Info(msg string, args ...interface{})  { l.log(INFO, "INFO ", msg, args...) }
func (l *Logger) Warn(msg string, args ...interface{})  { l.log(WARN, "WARN ", msg, args...) }
func (l *Logger) Error(msg string, args ...interface{}) { l.log(ERROR, "ERROR", msg, args...) }

func Debug(msg string, args ...interface{}) { Default.Debug(msg, args...) }
func Info(msg string, args ...interface{})  { Default.Info(msg, args...) }
func Warn(msg string, args ...interface{})  { Default.Warn(msg, args...) }
func Error(msg string, args ...interface{}) { Default.Error(msg, args...) }
