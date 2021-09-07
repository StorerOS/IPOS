package cmd

import (
	ring "container/ring"
	"context"
	"sync"

	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/cmd/ipos/logger/message/log"
	"github.com/storeros/ipos/cmd/ipos/logger/target/console"
	"github.com/storeros/ipos/pkg/pubsub"
)

const defaultLogBufferCount = 10000

type HTTPConsoleLoggerSys struct {
	sync.RWMutex
	pubsub   *pubsub.PubSub
	console  *console.Target
	nodeName string
	logBuf   *ring.Ring
}

func NewConsoleLogger(ctx context.Context) *HTTPConsoleLoggerSys {
	ps := pubsub.New()
	return &HTTPConsoleLoggerSys{
		pubsub:  ps,
		console: console.New(),
		logBuf:  ring.New(defaultLogBufferCount),
	}
}

func (sys *HTTPConsoleLoggerSys) HasLogListeners() bool {
	return sys != nil && sys.pubsub.HasSubscribers()
}

func (sys *HTTPConsoleLoggerSys) Subscribe(subCh chan interface{}, doneCh <-chan struct{}, node string, last int, logKind string, filter func(entry interface{}) bool) {
	if !sys.HasLogListeners() {
		logger.AddTarget(sys)
	}

	cnt := 0
	var lastN []log.Info
	if last > defaultLogBufferCount || last <= 0 {
		last = defaultLogBufferCount
	}

	lastN = make([]log.Info, last)
	sys.RLock()
	sys.logBuf.Do(func(p interface{}) {
		if p != nil {
			lg, ok := p.(log.Info)
			if ok && lg.SendLog(node, logKind) {
				lastN[cnt%last] = lg
				cnt++
			}
		}
	})
	sys.RUnlock()
	if cnt > 0 {
		for i := 0; i < last; i++ {
			entry := lastN[(cnt+i)%last]
			if (entry == log.Info{}) {
				continue
			}
			select {
			case subCh <- entry:
			case <-doneCh:
				return
			}
		}
	}
	sys.pubsub.Subscribe(subCh, doneCh, filter)
}

func (sys *HTTPConsoleLoggerSys) Send(e interface{}, logKind string) error {
	var lg log.Info
	switch e := e.(type) {
	case log.Entry:
		lg = log.Info{Entry: e, NodeName: sys.nodeName}
	case string:
		lg = log.Info{ConsoleMsg: e, NodeName: sys.nodeName}
	}

	sys.pubsub.Publish(lg)
	sys.Lock()
	sys.logBuf.Value = lg
	sys.logBuf = sys.logBuf.Next()
	sys.Unlock()

	return sys.console.Send(e, string(logger.All))
}
