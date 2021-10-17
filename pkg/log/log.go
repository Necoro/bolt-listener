package log

import (
	"fmt"
	"log"
	"os"
)

var debugLogger = log.New(os.Stdout, "DEBUG: ", log.LstdFlags|log.Lmsgprefix)
var verboseLogger = log.New(os.Stdout, " INFO: ", log.LstdFlags|log.Lmsgprefix)
var errorLogger = log.New(os.Stderr, "ERROR: ", log.LstdFlags|log.Lmsgprefix)
var warnLogger = log.New(os.Stdout, " WARN: ", log.LstdFlags|log.Lmsgprefix)

type logLevel byte

const (
	debug logLevel = iota
	info
	warn
)

var level logLevel = warn

func SetVerbose() {
	level = info
}

func SetDebug() {
	level = debug
}

func IsDebug() bool {
	return level == debug
}

func Debug(v ...interface{}) {
	if level <= debug {
		_ = debugLogger.Output(2, fmt.Sprint(v...))
	}
}

func Debugf(format string, v ...interface{}) {
	if level <= debug {
		_ = debugLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

func Print(v ...interface{}) {
	if level <= info {
		_ = verboseLogger.Output(2, fmt.Sprint(v...))
	}
}

func Printf(format string, v ...interface{}) {
	if level <= info {
		_ = verboseLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

func Error(v ...interface{}) {
	_ = errorLogger.Output(2, fmt.Sprint(v...))
}

//noinspection GoUnusedExportedFunction
func Errorf(format string, a ...interface{}) {
	_ = errorLogger.Output(2, fmt.Sprintf(format, a...))
}

//noinspection GoUnusedExportedFunction
func Warn(v ...interface{}) {
	_ = warnLogger.Output(2, fmt.Sprint(v...))
}

//noinspection GoUnusedExportedFunction
func Warnf(format string, a ...interface{}) {
	_ = warnLogger.Output(2, fmt.Sprintf(format, a...))
}
