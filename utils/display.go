package utils

import (
	"fmt"
	"sync"
	"time"
)

// RelayStatus represents the status of a relay operation
type RelayStatus struct {
	Name      string
	Status    string // "pending", "success", "cancelled", "error"
	Reason    string
	Duration  time.Duration
	StartTime time.Time
}

// StatusTracker tracks relay operation statuses
type StatusTracker struct {
	mu               sync.Mutex
	relays           map[string]*RelayStatus
	verbose          bool
	operationStart   time.Time
	firstResultTime  time.Duration
	firstResultReady bool
}

// NewStatusTracker creates a new status tracker
func NewStatusTracker(verbose bool) *StatusTracker {
	st := &StatusTracker{
		relays:           make(map[string]*RelayStatus),
		verbose:          verbose,
		operationStart:   time.Now(),
		firstResultReady: false,
	}
	return st
}

// AddRelay initializes a relay in the tracker
func (st *StatusTracker) AddRelay(name string) {
	if !st.verbose {
		return
	}
	st.mu.Lock()
	st.relays[name] = &RelayStatus{
		Name:      name,
		Status:    "pending",
		StartTime: time.Now(),
	}
	st.mu.Unlock()
}

// UpdateStatus updates the status of a relay
func (st *StatusTracker) UpdateStatus(name string, status string) {
	st.UpdateStatusWithReason(name, status, "")
}

// UpdateStatusWithReason updates the status of a relay with a reason
func (st *StatusTracker) UpdateStatusWithReason(name string, status string, reason string) {
	if !st.verbose {
		return
	}
	st.mu.Lock()
	if relay, exists := st.relays[name]; exists {
		relay.Status = status
		relay.Reason = reason
		relay.Duration = time.Since(relay.StartTime)
		// Record first successful result time
		if status == "success" && !st.firstResultReady {
			st.firstResultTime = time.Since(st.operationStart)
			st.firstResultReady = true
		}
	}
	st.mu.Unlock()
}

// FinalizeStatus ensures all non-successful relays have a reason
func (st *StatusTracker) FinalizeStatus() {
	if !st.verbose {
		return
	}
	st.mu.Lock()
	defer st.mu.Unlock()

	for _, relay := range st.relays {
		if relay.Status != "success" && relay.Reason == "" {
			if st.firstResultReady {
				relay.Reason = "cancelled by first result"
			} else {
				relay.Reason = "timeout"
			}
		}
	}
}

// DisplayStatus prints the current status of all relays
func (st *StatusTracker) DisplayStatus() {
	if !st.verbose {
		return
	}
	st.mu.Lock()
	defer st.mu.Unlock()

	for _, relay := range st.relays {
		icon := ""
		switch relay.Status {
		case "pending":
			icon = "⟳"
		case "success":
			icon = "✓"
		case "cancelled":
			icon = "✗"
		case "error":
			icon = "✗"
		}

		line := fmt.Sprintf("[%s] %-30s %dms", icon, relay.Name, relay.Duration.Milliseconds())
		if relay.Reason != "" {
			line += fmt.Sprintf(" (%s)", relay.Reason)
		}
		fmt.Println(line)
	}
}

// GetTotalDuration returns the time from operation start to first result
func (st *StatusTracker) GetTotalDuration() time.Duration {
	if !st.verbose {
		return 0
	}
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.firstResultReady {
		return st.firstResultTime
	}

	// If no successful result, return max (for compatibility)
	var maxDuration time.Duration
	for _, relay := range st.relays {
		if relay.Duration > maxDuration {
			maxDuration = relay.Duration
		}
	}
	return maxDuration
}
