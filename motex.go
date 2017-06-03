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

// Demote demotes m from a Lock() for writing to a lock for reading. It is a
// run-time error if m is not currently locked for writing. It is also a
// run-time error if m is locked for writing but demoted on entry to Demote.
//
// A demoted lock is not equivalent to an RLock. Only demoted locks can be
// promoted.
func (m *Motex) Demote() {
	m.r.Unlock()
	m.r.RLock()
}

// Promote promotes m from a Demoted() Lock() to a lock for writing. It is a
// run-time error if m is not currently a demoted lock.
//
// A demoted lock is not equivalent to an RLock. Only demoted locks can be
// promoted.
func (m *Motex) Promote() {
	m.r.RUnlock()
	m.r.Lock()
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
