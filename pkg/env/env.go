package env

import (
	"os"
	"strings"
	"sync"
)

var (
	privateMutex sync.RWMutex
	envOff       bool
)

func SetEnvOff() {
	privateMutex.Lock()
	defer privateMutex.Unlock()

	envOff = true
}

func SetEnvOn() {
	privateMutex.Lock()
	defer privateMutex.Unlock()

	envOff = false
}

func IsSet(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}

func Get(key, defaultValue string) string {
	privateMutex.RLock()
	ok := envOff
	privateMutex.RUnlock()
	if ok {
		return defaultValue
	}
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defaultValue
}

func List(prefix string) (envs []string) {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, prefix) {
			values := strings.SplitN(env, "=", 2)
			if len(values) == 2 {
				envs = append(envs, values[0])
			}
		}
	}
	return envs
}
