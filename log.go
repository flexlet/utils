package utils

import (
	"fmt"
	"log"
)

const (
	LOG_DEBUG int = iota
	LOG_INFO
	LOG_WARN
	LOG_ERROR
)

var (
	Logger   *log.Logger = log.Default()
	LogLevel int         = LOG_DEBUG
	LogLabel []string    = []string{"DEBUG", "INFO", "WARN", "ERROR"}
)

func LogPrintf(level int, module string, format string, v ...interface{}) {
	if level < LogLevel {
		return
	}
	Logger.Printf("[%s] %s: %s", LogLabel[level], module, fmt.Sprintf(format, v...))
}
