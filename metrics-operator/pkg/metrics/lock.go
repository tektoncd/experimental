package metrics

import (
	"sync"
	"time"
)

type Lock interface {
	LockWithExpire(value string, expire time.Duration)
	Release(value string)
}

type InMemoryLock struct {
	m  map[string]time.Duration
	rw sync.RWMutex
}

func (i *InMemoryLock) LockWithExpire(value string, expire time.Duration) {
	i.rw.Lock()
	defer i.rw.Unlock()
	i.m[value] = expire
}

func (i *InMemoryLock) Release(value string) {
	i.rw.Lock()
	defer i.rw.Unlock()
	delete(i.m, value)
}

func NewInMemoryLock() Lock {
	return &InMemoryLock{}
}
