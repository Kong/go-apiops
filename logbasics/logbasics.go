// This package provides a global logger instance that is used by other packages
// within go-apiops. The default is to discard any logging.
// It uses go-logr/logr. The logger instance can be set by calling SetLogger.
//
// General behaviour;
// * Errors will not be logged, but returned instead. Logging those is up to the caller.
// * The library does not use verbosity level 0
// * level 1 is used for informational messages (when calling `Info`)
// * level 2 is used for debug messages (when calling `Debug`)
package logbasics

import (
	"log"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

var (
	globalLogger  logr.Logger
	defaultLogger *logr.Logger
)

// Info logs an informational message ("info" at verbosity level 1).
func Info(msg string, keysAndValues ...interface{}) {
	globalLogger.V(1).Info(msg, keysAndValues...)
}

// Debug logs a debug message ("info" at verbosity level 2).
func Debug(msg string, keysAndValues ...interface{}) {
	globalLogger.V(2).Info(msg, keysAndValues...)
}

// Error logs an error message.
func Error(err error, msg string, keysAndValues ...interface{}) {
	globalLogger.Error(err, msg, keysAndValues...)
}

// SetLogger set the logger instance to use for logging.
// Setting it to nil will disable logging.
func SetLogger(l *logr.Logger) {
	if l == nil {
		globalLogger = logr.Discard()
	} else {
		globalLogger = *l
	}
}

// GetLogger returns the logger instance to use for logging.
func GetLogger() logr.Logger {
	return globalLogger
}

func newStderrLogger(flags int) stdr.StdLogger {
	return log.New(os.Stderr, "", flags)
}

// Initialize creates and sets a logger instance to log to stderr.
// Any follow up calls to Initialize will ignore the parameters and set the logger to
// the initially created instance.
// see https://pkg.go.dev/log#pkg-constants for the flag values
func Initialize(flags int, verbosity int) {
	if defaultLogger == nil {
		stdr.SetVerbosity(verbosity)
		l := stdr.New(newStderrLogger(flags))
		defaultLogger = &l
	}
	SetLogger(defaultLogger)
}

func init() {
	SetLogger(nil) // discard logs by default
}
