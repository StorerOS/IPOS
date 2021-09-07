package cmd

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	dynamicTimeoutIncreaseThresholdPct = 0.33
	dynamicTimeoutDecreaseThresholdPct = 0.10
	dynamicTimeoutLogSize              = 16
	maxDuration                        = time.Duration(1<<63 - 1)
)

type dynamicTimeout struct {
	timeout int64
	minimum int64
	entries int64
	log     [dynamicTimeoutLogSize]time.Duration
	mutex   sync.Mutex
}

func newDynamicTimeout(timeout, minimum time.Duration) *dynamicTimeout {
	return &dynamicTimeout{timeout: int64(timeout), minimum: int64(minimum)}
}

func (dt *dynamicTimeout) Timeout() time.Duration {
	return time.Duration(atomic.LoadInt64(&dt.timeout))
}

func (dt *dynamicTimeout) LogSuccess(duration time.Duration) {
	dt.logEntry(duration)
}

func (dt *dynamicTimeout) LogFailure() {
	dt.logEntry(maxDuration)
}

func (dt *dynamicTimeout) logEntry(duration time.Duration) {
	entries := int(atomic.AddInt64(&dt.entries, 1))
	index := entries - 1
	if index < dynamicTimeoutLogSize {
		dt.mutex.Lock()
		dt.log[index] = duration
		dt.mutex.Unlock()
	}
	if entries == dynamicTimeoutLogSize {
		dt.mutex.Lock()

		logCopy := [dynamicTimeoutLogSize]time.Duration{}
		copy(logCopy[:], dt.log[:])

		atomic.StoreInt64(&dt.entries, 0)

		dt.mutex.Unlock()

		dt.adjust(logCopy)
	}
}

func (dt *dynamicTimeout) adjust(entries [dynamicTimeoutLogSize]time.Duration) {

	failures, average := 0, int64(0)
	for i := 0; i < len(entries); i++ {
		if entries[i] == maxDuration {
			failures++
		} else {
			average += int64(entries[i])
		}
	}
	if failures < len(entries) {
		average /= int64(len(entries) - failures)
	}

	timeOutHitPct := float64(failures) / float64(len(entries))

	if timeOutHitPct > dynamicTimeoutIncreaseThresholdPct {
		timeout := atomic.LoadInt64(&dt.timeout) * 125 / 100
		atomic.StoreInt64(&dt.timeout, timeout)
	} else if timeOutHitPct < dynamicTimeoutDecreaseThresholdPct {
		average = average * 125 / 100

		timeout := (atomic.LoadInt64(&dt.timeout) + int64(average)) / 2
		if timeout < dt.minimum {
			timeout = dt.minimum
		}
		atomic.StoreInt64(&dt.timeout, timeout)
	}

}
