package logger

import (
	"context"
	"sync"

	"time"
)

type logOnceType struct {
	IDMap map[interface{}]error
	sync.Mutex
}

func (l *logOnceType) logOnceIf(ctx context.Context, err error, id interface{}, errKind ...interface{}) {
	if err == nil {
		return
	}
	l.Lock()
	shouldLog := false
	prevErr := l.IDMap[id]
	if prevErr == nil {
		l.IDMap[id] = err
		shouldLog = true
	} else {
		if prevErr.Error() != err.Error() {
			l.IDMap[id] = err
			shouldLog = true
		}
	}
	l.Unlock()

	if shouldLog {
		LogIf(ctx, err, errKind...)
	}
}

func (l *logOnceType) cleanupRoutine() {
	for {
		l.Lock()
		l.IDMap = make(map[interface{}]error)
		l.Unlock()

		time.Sleep(30 * time.Minute)
	}
}

func newLogOnceType() *logOnceType {
	l := &logOnceType{IDMap: make(map[interface{}]error)}
	go l.cleanupRoutine()
	return l
}

var logOnce = newLogOnceType()

func LogOnceIf(ctx context.Context, err error, id interface{}, errKind ...interface{}) {
	logOnce.logOnceIf(ctx, err, id, errKind...)
}
