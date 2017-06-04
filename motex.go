// Package motex implements a demoteable and promoteable exclusion lock.
// Demoteable meaning a write lock can be changed to a read lock. Promoteable
// meaning a demoted write lock can be changed back to a write lock.
//
// If Demote() and Promote() are never called, motex behaves identically to a
// sync.RWMutex.
//
// A demoted lock continues to block calls to Lock, but calls to RLock will not
// block.
//
// non-atomic read and update with sync.Mutex:
//  var mu sync.Mutex
//  var state int
//  mu.RLock()
//  copied := state
//  mu.RUnlock()
//  copied += 1 // ... do something resource intensive to modify the copied state.
//  mu.Lock()
//  state = copied
//  mu.Unlock()
//
// atomic read and update with motex.Motex:
//  var mo motex.Motex
//  var state int
//  mo.Lock()
//  mo.Demote()
//  copied := state
//  copied += 1 // ... do something resource intensive to modiy the copied state.
//  mo.Promote()
//  state = copied
//  mo.Unlock()
package motex

import (
	"sync"
)

// A Motex is a demoteable and promoteable exclusion lock.
//
// A Motex must not be copied after first use.
type Motex struct {
	w sync.Mutex   // Blocks writes.
	r sync.RWMutex // Blocks reads.

	g sync.Mutex // Guards against Promote after RLock.
}

// Lock locks m for writing. If the lock is already locked for reading or
// writing, Lock blocks until the lock is available (i.e. after Unlock or
// RUnlock is called).
func (m *Motex) Lock() {
	m.w.Lock()
	m.r.Lock()
}

// Unlock unlocks m for writing. It is a run-time error if m is not locked for
// writing on entry to Unlock. It is also a run-time error if m is locked for
// writing but demoted on entry to Unlock.
func (m *Motex) Unlock() {
	m.r.Unlock()
	m.w.Unlock()
}

// Demote demotes m from a Lock() for writing to a lock for reading, unblocking
// any calls to RLock. Other calls to Lock will continue to block. It is a
// run-time error if m is not currently locked for writing. It is also a
// run-time error if m is locked for writing but demoted on entry to Demote.
//
// A demoted lock is not equivalent to an RLock. Only demoted locks can be
// promoted.
func (m *Motex) Demote() {
	m.r.Unlock()
	m.r.RLock()

	m.g.Lock()
}

// Promote promotes m from a Demoted() Lock() to a lock for writing, blocking
// all calls to RLock. If the lock is already RLocked, Promote blocks until the
// lock is available (i.e. after RUnlock is called). It is a run-time error if
// m is not currently a demoted lock.
//
// A demoted lock is not equivalent to an RLock. Only demoted locks can be
// promoted.
func (m *Motex) Promote() {
	m.r.RUnlock()
	m.r.Lock()

	m.g.Unlock()
}

// RLock locks m for reading. If the lock is already locked for writing or
// demoted reading, RLock blocks until the lock is available (i.e. after Unlock
// is called).
func (m *Motex) RLock() {
	m.r.RLock()
}

// RUnlock unlocks m for reading. It is a run-time error if m is not locked for
// reading on entry to RUnlock.
func (m *Motex) RUnlock() {
	m.r.RUnlock()
}
