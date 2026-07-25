package logger

import (
	"log"
	"os"
	"sync"
)

type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	debugMode   bool
	mu          sync.RWMutex
}

var (
	instance *Logger
	once     sync.Once
)

func Init(debugMode bool) *Logger {
	once.Do(func() {
		instance = &Logger{
			infoLogger:  log.New(os.Stdout, "[INFO] ", log.LstdFlags|log.Lmsgprefix),
			errorLogger: log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lmsgprefix),
			debugLogger: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lmsgprefix),
			debugMode:   debugMode,
		}
	})
	return instance
}

func Get() *Logger {
	if instance == nil {
		return Init(false)
	}
	return instance
}

func (l *Logger) SetDebugMode(debug bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debugMode = debug
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.infoLogger.Printf(format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.errorLogger.Printf(format, v...)
}

func (l *Logger) Debug(format string, v ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.debugMode {
		l.debugLogger.Printf(format, v...)
	}
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.infoLogger.Printf(format, v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.errorLogger.Printf(format, v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.debugMode {
		l.debugLogger.Printf(format, v...)
	}
}

func Info(format string, v ...interface{}) {
	Get().Info(format, v...)
}

func Error(format string, v ...interface{}) {
	Get().Error(format, v...)
}

func Debug(format string, v ...interface{}) {
	Get().Debug(format, v...)
}