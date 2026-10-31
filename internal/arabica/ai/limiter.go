package ai

import (
	"sync"
	"time"
)

// maxActiveRequests bounds simultaneous provider calls and library loads globally.
const maxActiveRequests = 8

type admission struct {
	start  time.Time
	count  int
	active bool
}
type limiter struct {
	mu       sync.Mutex
	accounts map[string]admission
	active   int
}

func (l *limiter) acquire(did string, now time.Time) (func(), error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.accounts == nil {
		l.accounts = make(map[string]admission)
	}
	for key, state := range l.accounts {
		if !state.active && now.Sub(state.start) >= time.Minute {
			delete(l.accounts, key)
		}
	}
	if l.active >= maxActiveRequests {
		return nil, ErrLimit
	}
	state, exists := l.accounts[did]
	if !exists && len(l.accounts) >= 1024 {
		return nil, ErrLimit
	}
	if state.active || state.count >= 10 {
		return nil, ErrLimit
	}
	if !exists {
		state.start = now
	}
	state.count++
	state.active = true
	l.active++
	l.accounts[did] = state
	return func() {
		l.mu.Lock()
		s := l.accounts[did]
		s.active = false
		l.accounts[did] = s
		l.active--
		l.mu.Unlock()
	}, nil
}
