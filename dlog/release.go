// +build !debug

package dlog

// Debugf no-op for release builds
func (l *Logger) Debugf(format string, v ...interface{}) {}

// Debug no-op for release builds
func (l *Logger) Debug(v ...interface{}) {}

// Debugln no-op for release builds
func (l *Logger) Debugln(v ...interface{}) {}
