package logging

import (
	"io"
	"log"
	"os"
)

// Level is the type for log levels.
type Level int

const (
	// LevelDebug means log debug messages and above.
	LevelDebug Level = iota
	// LevelInfo means log info messages and above.
	LevelInfo
	// LevelWarning means log warning messages and above.
	LevelWarning
	// LevelError means log error messages only.
	LevelError
	// LevelNone means log nothing.
	LevelNone
)

var (
	debug   *log.Logger
	info    *log.Logger
	warning *log.Logger
	error   *log.Logger
)

func init() {
	flags := log.Ldate | log.Ltime | log.LUTC
	debug = log.New(io.Discard, "D ", flags)
	info = log.New(io.Discard, "I ", flags)
	warning = log.New(io.Discard, "W ", flags)
	error = log.New(io.Discard, "E ", flags)

	SetLevel(LevelWarning)
}

// SetLevel sets the log level.
func SetLevel(l Level) {
	switch l {
	case LevelDebug:
		debug.SetOutput(os.Stderr)
		info.SetOutput(os.Stderr)
		warning.SetOutput(os.Stderr)
		error.SetOutput(os.Stderr)
	case LevelInfo:
		debug.SetOutput(io.Discard)
		info.SetOutput(os.Stderr)
		warning.SetOutput(os.Stderr)
		error.SetOutput(os.Stderr)
	case LevelWarning:
		debug.SetOutput(io.Discard)
		info.SetOutput(io.Discard)
		warning.SetOutput(os.Stderr)
		error.SetOutput(os.Stderr)
	case LevelError:
		debug.SetOutput(io.Discard)
		info.SetOutput(io.Discard)
		warning.SetOutput(io.Discard)
		error.SetOutput(os.Stderr)
	case LevelNone:
		debug.SetOutput(io.Discard)
		info.SetOutput(io.Discard)
		warning.SetOutput(io.Discard)
		error.SetOutput(io.Discard)
	}
}

// Debug logs a debug message.
func Debug(msg string, v ...interface{}) {
	debug.Printf(msg, v...)
}

// Info logs a message with level info.
func Info(msg string, v ...interface{}) {
	info.Printf(msg, v...)
}

// Warning logs a message with level warning.
func Warning(msg string, v ...interface{}) {
	warning.Printf(msg, v...)
}

// Error logs a message with level error.
func Error(msg string, v ...interface{}) {
	error.Printf(msg, v...)
}
