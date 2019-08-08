package dlog

import (
	"io"
	"log"
	"os"
)

type Logger struct {
	*log.Logger
}

type LoggerOption struct {
	f func(*Logger)
}

// NewLogger is a simple wrapper around the default Go log package
// It adds Debug functions to add a layer of logging that is useful
// for development purposes; these logs can be enabled with a build
// flag of //+build debug - they are otherwise disabled by default
func NewLogger(options ...LoggerOption) *Logger {
	l := &Logger{log.New(os.Stderr, "", log.LstdFlags)}

	for _, option := range options {
		option.f(l)
	}

	return l
}

func LoggerSetOutput(w io.Writer) LoggerOption {
	return LoggerOption{
		func(l *Logger) {
			l.SetOutput(w)
		},
	}
}

func LoggerSetPrefix(p string) LoggerOption {
	return LoggerOption{
		func(l *Logger) {
			l.SetPrefix(p)
		},
	}
}

func LoggerSetFlags(flag int) LoggerOption {
	return LoggerOption{
		func(l *Logger) {
			l.SetFlags(flag)
		},
	}
}
