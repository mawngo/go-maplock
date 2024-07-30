package maplock

import "sync"

import (
	"sync/atomic"
)

// MapLock provides a locking mechanism based on the passed in reference name.
type MapLock[T comparable] struct {
	mu    sync.Mutex
	locks map[T]*lockCtr
}

// lockCtr is used by MapLock to represent a lock with a given name.
type lockCtr struct {
	mu sync.Mutex
	// waiters is the number of waiters waiting to acquire the lock
	// this is int32 instead of uint32, so we can add `-1` in `dec()`
	waiters int32
}

// inc increments the number of waiters waiting for the lock.
func (l *lockCtr) inc() {
	atomic.AddInt32(&l.waiters, 1)
}

// dec decrements the number of waiters waiting on the lock.
func (l *lockCtr) dec() {
	atomic.AddInt32(&l.waiters, -1)
}

// count gets the current number of waiters.
func (l *lockCtr) count() int32 {
	return atomic.LoadInt32(&l.waiters)
}

// Lock locks the mutex.
func (l *lockCtr) Lock() {
	l.mu.Lock()
}

// Unlock unlocks the mutex.
func (l *lockCtr) Unlock() {
	l.mu.Unlock()
}

// TryLock tries to lock m and reports whether it succeeded.
func (l *lockCtr) TryLock() bool {
	return l.mu.TryLock()
}

// New creates a new MapLock.
func New[T comparable]() *MapLock[T] {
	return &MapLock[T]{
		locks: make(map[T]*lockCtr),
	}
}

// Lock locks a mutex with the given name. If it doesn't exist, one is created.
func (l *MapLock[T]) Lock(name T) {
	l.mu.Lock()
	if l.locks == nil {
		l.locks = make(map[T]*lockCtr)
	}

	nameLock, exists := l.locks[name]
	if !exists {
		nameLock = &lockCtr{}
		l.locks[name] = nameLock
	}

	// increment the nameLock waiters while inside the main mutex
	// this makes sure that the lock isn't deleted if `Lock` and `Unlock` are called concurrently.
	nameLock.inc()
	l.mu.Unlock()

	// Lock the nameLock outside the main mutex, so we don't block other operations
	// once locked then we can decrement the number of waiters for this lock.
	nameLock.Lock()
	nameLock.dec()
}

// TryLock tries to lock a mutex with the given name and reports whether it succeeded.
func (l *MapLock[T]) TryLock(name T) bool {
	l.mu.Lock()
	if l.locks == nil {
		l.locks = make(map[T]*lockCtr)
	}

	nameLock, exists := l.locks[name]
	if !exists {
		nameLock = &lockCtr{}
		l.locks[name] = nameLock
	}

	// increment the nameLock waiters while inside the main mutex
	// this makes sure that the lock isn't deleted if `Lock` and `Unlock` are called concurrently.
	nameLock.inc()
	try := l.mu.TryLock()

	// Lock the nameLock outside the main mutex, so we don't block other operations
	// once locked then we can decrement the number of waiters for this lock.
	nameLock.Lock()
	nameLock.dec()
	return try
}

// Unlock unlocks the mutex with the given name
// If the given lock is not being waited on by any other callers, it is deleted.
func (l *MapLock[T]) Unlock(name T) {
	l.mu.Lock()
	nameLock, exists := l.locks[name]
	if !exists {
		l.mu.Unlock()
		panic("unlock of unlocked entry")
	}

	if nameLock.count() == 0 {
		delete(l.locks, name)
	}
	nameLock.Unlock()

	l.mu.Unlock()
}
