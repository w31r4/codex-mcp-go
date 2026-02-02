package mcp

import (
	"context"
	"sync"
	"time"
)

type workdirLockMode string

const (
	workdirLockReject workdirLockMode = "reject"
	workdirLockQueue  workdirLockMode = "queue"
)

type workdirLockManager struct {
	mu    sync.Mutex
	locks map[string]chan struct{}
}

func newWorkdirLockManager() *workdirLockManager {
	return &workdirLockManager{
		locks: make(map[string]chan struct{}),
	}
}

func (m *workdirLockManager) acquire(ctx context.Context, key string, mode workdirLockMode, timeout time.Duration) (bool, error) {
	if m == nil || key == "" {
		return true, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	ch := m.getOrCreate(key)

	switch mode {
	case workdirLockQueue:
		if timeout > 0 {
			t := time.NewTimer(timeout)
			defer t.Stop()
			select {
			case ch <- struct{}{}:
				return true, nil
			case <-t.C:
				return false, nil
			case <-ctx.Done():
				return false, ctx.Err()
			}
		}
		select {
		case ch <- struct{}{}:
			return true, nil
		case <-ctx.Done():
			return false, ctx.Err()
		}
	case workdirLockReject:
		fallthrough
	default:
		select {
		case ch <- struct{}{}:
			return true, nil
		default:
			return false, nil
		}
	}
}

func (m *workdirLockManager) release(key string) {
	if m == nil || key == "" {
		return
	}

	ch := m.getOrCreate(key)
	select {
	case <-ch:
	default:
	}
}

func (m *workdirLockManager) getOrCreate(key string) chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch, ok := m.locks[key]
	if ok {
		return ch
	}
	ch = make(chan struct{}, 1)
	m.locks[key] = ch
	return ch
}
