package log

import (
	log "log"
	"sync"
)

// Logger represents a collection of log.Logger's to log messages and/or
// errors to. This typically includes stdout/err and access/error log files
type Logger struct {
	logs   []*log.Logger
	errors []*log.Logger
	mutex  sync.RWMutex
	emutex sync.RWMutex
}

// Log passes the given arguments to the Print() function of each non-error
// log.Logger it contains
func (l Logger) Log(v ...interface{}) {
	l.mutex.RLock()
	for _, lg := range l.logs {
		lg.Print(v...)
	}
	l.mutex.RUnlock()
}

// Logf passes the given arguments to the Printf() function of each non-error
// log.Logger it contains
func (l Logger) Logf(format string, v ...interface{}) {
	l.mutex.RLock()
	for _, lg := range l.logs {
		lg.Printf(format, v...)
	}
	l.mutex.RUnlock()
}

// Logln passes the given arguments to the Println() function of each non-error
// log.Logger it contains
func (l Logger) Logln(v ...interface{}) {
	l.mutex.RLock()
	for _, lg := range l.logs {
		lg.Println(v...)
	}
	l.mutex.RUnlock()
}

// Error passes the given arguments to the Print() function of each error
// log.Logger it contains
func (l Logger) Error(v ...interface{}) {
	l.emutex.RLock()
	for _, err := range l.errors {
		err.Print(v...)
	}
	l.emutex.RUnlock()
}

// Errorf passes the given arguments to the Printf() function of each error
// log.Logger it contains
func (l Logger) Errorf(format string, v ...interface{}) {
	l.emutex.RLock()
	for _, err := range l.errors {
		err.Printf(format, v...)
	}
	l.emutex.RUnlock()
}

// Errorln passes the given arguments to the Println() function of each error
// log.Logger it contains
func (l Logger) Errorln(v ...interface{}) {
	l.emutex.RLock()
	for _, err := range l.errors {
		err.Println(v...)
	}
	l.emutex.RUnlock()
}

// AddLogger adds a non-error log.Logger to this Logger's collection
func (l *Logger) AddLogger(lg *log.Logger) {
	l.mutex.Lock()
	l.logs = append(l.logs, lg)
	l.mutex.Unlock()
}

// AddErrorLogger adds an error log.Logger to this Logger's collection
func (l *Logger) AddErrorLogger(lg *log.Logger) {
	l.emutex.Lock()
	l.errors = append(l.errors, lg)
	l.emutex.Unlock()
}
