package logging

import (
	"io/ioutil"
	"log"
	"os"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarning
	LevelError
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
	debug = log.New(ioutil.Discard, "D ", flags)
	info = log.New(ioutil.Discard, "I ", flags)
	warning = log.New(ioutil.Discard, "W ", flags)
	error = log.New(ioutil.Discard, "E ", flags)

	SetLevel(LevelWarning)
}

func SetLevel(l Level) {
	switch l {
	case LevelDebug:
		debug.SetOutput(os.Stderr)
		info.SetOutput(os.Stderr)
		warning.SetOutput(os.Stderr)
		error.SetOutput(os.Stderr)
	case LevelInfo:
		debug.SetOutput(ioutil.Discard)
		info.SetOutput(os.Stderr)
		warning.SetOutput(os.Stderr)
		error.SetOutput(os.Stderr)
	case LevelWarning:
		debug.SetOutput(ioutil.Discard)
		info.SetOutput(ioutil.Discard)
		warning.SetOutput(os.Stderr)
		error.SetOutput(os.Stderr)
	case LevelError:
		debug.SetOutput(ioutil.Discard)
		info.SetOutput(ioutil.Discard)
		warning.SetOutput(ioutil.Discard)
		error.SetOutput(os.Stderr)
	case LevelNone:
		debug.SetOutput(ioutil.Discard)
		info.SetOutput(ioutil.Discard)
		warning.SetOutput(ioutil.Discard)
		error.SetOutput(ioutil.Discard)
	}
}

func Debug(msg string, v ...interface{}) {
	debug.Printf(msg, v...)
}

func Info(msg string, v ...interface{}) {
	info.Printf(msg, v...)
}

func Warning(msg string, v ...interface{}) {
	warning.Printf(msg, v...)
}

func Error(msg string, v ...interface{}) {
	error.Printf(msg, v...)
}
