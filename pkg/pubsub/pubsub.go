package pubsub

import (
	"sync"
)

type Sub struct {
	ch     chan interface{}
	filter func(entry interface{}) bool
}

type PubSub struct {
	subs []*Sub
	sync.RWMutex
}

func (ps *PubSub) Publish(item interface{}) {
	ps.RLock()
	defer ps.RUnlock()

	for _, sub := range ps.subs {
		if sub.filter == nil || sub.filter(item) {
			select {
			case sub.ch <- item:
			default:
			}
		}
	}
}

func (ps *PubSub) Subscribe(subCh chan interface{}, doneCh <-chan struct{}, filter func(entry interface{}) bool) {
	ps.Lock()
	defer ps.Unlock()

	sub := &Sub{subCh, filter}
	ps.subs = append(ps.subs, sub)

	go func() {
		<-doneCh

		ps.Lock()
		defer ps.Unlock()

		for i, s := range ps.subs {
			if s == sub {
				ps.subs = append(ps.subs[:i], ps.subs[i+1:]...)
			}
		}
	}()
}

func (ps *PubSub) HasSubscribers() bool {
	ps.RLock()
	defer ps.RUnlock()
	return len(ps.subs) > 0
}

func New() *PubSub {
	return &PubSub{}
}
