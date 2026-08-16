package oncall

import (
	"fmt"
	"sort"
	"time"
)

// DateLayout is the date format used throughout the package.
const DateLayout = "2006-01-02"

// Build computes the on-call schedule for the given request.
//
// The rotation maintains a pointer into Roster. The pointer advances only on
// days an engineer is actually assigned: it moves to (assignedIndex + 1) mod
// len(Roster). On holidays, when the roster is empty, or when every engineer
// is blacked out, the pointer is frozen so the next business day retries from
// the same position.
func Build(req Request) (*Schedule, error) {
	start, err := time.Parse(DateLayout, req.Start)
	if err != nil {
		return nil, fmt.Errorf("invalid start date %q: %w", req.Start, err)
	}
	end, err := time.Parse(DateLayout, req.End)
	if err != nil {
		return nil, fmt.Errorf("invalid end date %q: %w", req.End, err)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("end date %s is before start date %s", req.End, req.Start)
	}

	n := len(req.Roster)
	if n > 0 && (req.StartIndex < 0 || req.StartIndex >= n) {
		return nil, fmt.Errorf("start index %d out of range [0,%d)", req.StartIndex, n)
	}

	entries := make([]ScheduleEntry, 0, daysBetween(start, end)+1)
	fairness := make(map[string]int)
	pointer := req.StartIndex

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		ds := d.Format(DateLayout)
		entry := ScheduleEntry{Date: ds, Weekday: d.Weekday().String()}

		switch {
		case req.Holidays[ds]:
			// Holiday: no one is on call and the pointer is frozen.
			entry.Status = StatusHoliday
		case n == 0:
			// Empty roster: nothing can be covered.
			entry.Status = StatusUncovered
		default:
			found := -1
			for i := 0; i < n; i++ {
				idx := (pointer + i) % n
				if !isBlackout(req.Blackouts, req.Roster[idx], ds) {
					found = idx
					break
				}
			}
			if found == -1 {
				// Everyone is blacked out: leave uncovered and freeze the
				// pointer so the next day retries from the same position.
				entry.Status = StatusUncovered
			} else {
				entry.Status = StatusAssigned
				entry.Engineer = req.Roster[found]
				fairness[req.Roster[found]]++
				pointer = (found + 1) % n
			}
		}
		entries = append(entries, entry)
	}

	sched := &Schedule{Entries: entries}
	for eng, days := range fairness {
		sched.Fairness = append(sched.Fairness, FairnessCount{Engineer: eng, Days: days})
	}
	sort.Slice(sched.Fairness, func(i, j int) bool {
		return sched.Fairness[i].Engineer < sched.Fairness[j].Engineer
	})
	return sched, nil
}

// isBlackout reports whether eng cannot serve on date.
func isBlackout(bl map[string]map[string]bool, eng, date string) bool {
	if bl == nil {
		return false
	}
	days, ok := bl[eng]
	if !ok {
		return false
	}
	return days[date]
}

// daysBetween returns the whole-day count from start to end (inclusive of end).
// It is only used to size slice capacity, so it need not be exact across DST
// transitions; date-only parsing yields UTC midnight, so there is no DST.
func daysBetween(start, end time.Time) int {
	return int(end.Sub(start).Hours() / 24)
}
