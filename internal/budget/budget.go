// Package budget tracks API spending and enforces cost limits.
// Monitors token usage across providers and stops work when limits hit.
package budget

// Tracker monitors spending against configured limits.
type Tracker struct {
	// TODO: Add fields for tracking spend per provider
}

// NewTracker creates a budget tracker with the given limit.
func NewTracker(limitCents int64) *Tracker {
	// TODO: Implement
	return &Tracker{}
}

// Record logs spending for a provider.
func (t *Tracker) Record(provider string, tokens int, costCents int64) {
	// TODO: Implement
}

// Remaining returns cents left in budget.
func (t *Tracker) Remaining() int64 {
	// TODO: Implement
	return 0
}
