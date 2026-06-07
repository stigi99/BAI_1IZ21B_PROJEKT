package security

import "sync/atomic"

// ModeStore keeps the lab security mode in one concurrency-safe place.
type ModeStore struct {
	enabled atomic.Bool
}

// NewModeStore creates a mode store initialized with the provided value.
//
// Input:
//   - initial: starting security state for the lab.
//
// Output:
//   - *ModeStore: concurrency-safe holder for the current mode.
func NewModeStore(initial bool) *ModeStore {
	store := &ModeStore{}
	store.enabled.Store(initial)
	return store
}

// Enabled reports the current security mode.
//
// Output:
//   - bool: true when secure mode is active.
func (s *ModeStore) Enabled() bool {
	if s == nil {
		return false
	}
	return s.enabled.Load()
}

// Set overwrites the current security mode with the provided value.
//
// Input:
//   - enabled: new security state to store.
func (s *ModeStore) Set(enabled bool) {
	if s == nil {
		return
	}
	s.enabled.Store(enabled)
}

// Toggle flips the current security mode and returns the new value.
//
// Output:
//   - bool: updated security state after the toggle completes.
func (s *ModeStore) Toggle() bool {
	if s == nil {
		return false
	}
	for {
		current := s.enabled.Load()
		next := !current
		if s.enabled.CompareAndSwap(current, next) {
			return next
		}
	}
}
