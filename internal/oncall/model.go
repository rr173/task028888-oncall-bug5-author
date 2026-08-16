// Package oncall implements an on-call rotation scheduler.
//
// The scheduler walks a date range day by day, assigning one engineer per
// business day from an ordered roster. Holidays pause the rotation and
// per-engineer blackout days cause that engineer to be skipped, with the
// rotation resuming from the engineer after the one actually assigned.
package oncall

// DayStatus describes the on-call state of a single day.
type DayStatus string

const (
	// StatusAssigned means an engineer is on call for the day.
	StatusAssigned DayStatus = "assigned"
	// StatusHoliday means the day is a holiday; no one is on call.
	StatusHoliday DayStatus = "holiday"
	// StatusUncovered means no engineer could cover the day.
	StatusUncovered DayStatus = "uncovered"
)

// ScheduleEntry is one day's on-call assignment.
type ScheduleEntry struct {
	Date     string // YYYY-MM-DD
	Weekday  string // e.g. Monday
	Status   DayStatus
	Engineer string // non-empty only when Status == StatusAssigned
}

// FairnessCount reports how many on-call days an engineer served.
type FairnessCount struct {
	Engineer string
	Days     int
}

// Schedule is the full rotation result.
type Schedule struct {
	Entries  []ScheduleEntry
	Fairness []FairnessCount
}

// Request is the input to Build.
type Request struct {
	// Roster is the ordered list of engineer IDs. May be empty, in which
	// case every business day is uncovered.
	Roster []string
	// Start is the inclusive start date (YYYY-MM-DD).
	Start string
	// End is the inclusive end date (YYYY-MM-DD). Must be >= Start.
	End string
	// StartIndex is the 0-based roster index used for the first business
	// day. Must be in [0, len(Roster)) when Roster is non-empty.
	StartIndex int
	// Holidays is the set of holiday dates (YYYY-MM-DD). On a holiday the
	// rotation pointer does not advance.
	Holidays map[string]bool
	// Blackouts maps an engineer ID to the set of dates they cannot serve.
	Blackouts map[string]map[string]bool
}
