package oncall

import (
	"testing"
)

// statusOrEngineer renders an entry for compact comparison: the assigned
// engineer's ID, or the status string otherwise.
func statusOrEngineer(e ScheduleEntry) string {
	if e.Status == StatusAssigned {
		return e.Engineer
	}
	return string(e.Status)
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func entriesAsStrings(s *Schedule) []string {
	out := make([]string, 0, len(s.Entries))
	for _, e := range s.Entries {
		out = append(out, statusOrEngineer(e))
	}
	return out
}

func TestBuild_PlainRoundRobin(t *testing.T) {
	s, err := Build(Request{
		Roster:     []string{"A", "B", "C"},
		Start:      "2026-03-02",
		End:        "2026-03-07",
		StartIndex: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"A", "B", "C", "A", "B", "C"}
	if got := entriesAsStrings(s); !equalStringSlice(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
	wantFair := []FairnessCount{{"A", 2}, {"B", 2}, {"C", 2}}
	if !equalFairness(s.Fairness, wantFair) {
		t.Errorf("fairness = %+v want %+v", s.Fairness, wantFair)
	}
}

func TestBuild_HolidayPause(t *testing.T) {
	s, err := Build(Request{
		Roster:     []string{"A", "B", "C"},
		Start:      "2026-03-02",
		End:        "2026-03-06",
		StartIndex: 0,
		Holidays:   map[string]bool{"2026-03-03": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 03-02 A (ptr->1), 03-03 holiday (ptr frozen 1), 03-04 B (ptr->2),
	// 03-05 C (ptr->0), 03-06 A (ptr->1).
	want := []string{"A", "holiday", "B", "C", "A"}
	if got := entriesAsStrings(s); !equalStringSlice(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestBuild_BlackoutSkip(t *testing.T) {
	s, err := Build(Request{
		Roster:     []string{"A", "B", "C"},
		Start:      "2026-03-02",
		End:        "2026-03-04",
		StartIndex: 0,
		Blackouts:  map[string]map[string]bool{"A": {"2026-03-02": true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 03-02 A blacked out -> B (ptr->2), 03-03 C (ptr->0), 03-04 A (ptr->1).
	want := []string{"B", "C", "A"}
	if got := entriesAsStrings(s); !equalStringSlice(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestBuild_AllBlackoutUncovered(t *testing.T) {
	s, err := Build(Request{
		Roster:     []string{"A"},
		Start:      "2026-03-02",
		End:        "2026-03-03",
		StartIndex: 0,
		Blackouts:  map[string]map[string]bool{"A": {"2026-03-02": true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 03-02 uncovered (ptr frozen 0), 03-03 A.
	want := []string{"uncovered", "A"}
	if got := entriesAsStrings(s); !equalStringSlice(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestBuild_EmptyRoster(t *testing.T) {
	s, err := Build(Request{
		Roster: nil,
		Start:  "2026-03-02",
		End:    "2026-03-03",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"uncovered", "uncovered"}
	if got := entriesAsStrings(s); !equalStringSlice(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
	if len(s.Fairness) != 0 {
		t.Errorf("fairness = %+v, want empty", s.Fairness)
	}
}

func TestBuild_StartIndexNonZero(t *testing.T) {
	s, err := Build(Request{
		Roster:     []string{"A", "B", "C"},
		Start:      "2026-03-02",
		End:        "2026-03-04",
		StartIndex: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"C", "A", "B"}
	if got := entriesAsStrings(s); !equalStringSlice(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestBuild_HolidayAtStartAndEnd(t *testing.T) {
	s, err := Build(Request{
		Roster:     []string{"A", "B"},
		Start:      "2026-03-02",
		End:        "2026-03-05",
		StartIndex: 0,
		Holidays:   map[string]bool{"2026-03-02": true, "2026-03-05": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 03-02 holiday (ptr 0), 03-03 A (ptr->1), 03-04 B (ptr->0), 03-05 holiday (ptr 0).
	want := []string{"holiday", "A", "B", "holiday"}
	if got := entriesAsStrings(s); !equalStringSlice(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestBuild_SingleDayRange(t *testing.T) {
	s, err := Build(Request{
		Roster:     []string{"A", "B"},
		Start:      "2026-03-02",
		End:        "2026-03-02",
		StartIndex: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"B"}
	if got := entriesAsStrings(s); !equalStringSlice(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestBuild_BlackoutWrapAround(t *testing.T) {
	// A is blacked out on 03-03 only; rotation should continue past it.
	s, err := Build(Request{
		Roster:     []string{"A", "B", "C"},
		Start:      "2026-03-02",
		End:        "2026-03-05",
		StartIndex: 0,
		Blackouts:  map[string]map[string]bool{"A": {"2026-03-03": true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 03-02 A (ptr->1), 03-03 B (ptr->2), 03-04 C (ptr->0), 03-05 A (ptr->1).
	want := []string{"A", "B", "C", "A"}
	if got := entriesAsStrings(s); !equalStringSlice(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestBuild_FairnessSortedAlphabetically(t *testing.T) {
	s, err := Build(Request{
		Roster:     []string{"carol", "alice", "bob"},
		Start:      "2026-03-02",
		End:        "2026-03-04",
		StartIndex: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []FairnessCount{{"alice", 1}, {"bob", 1}, {"carol", 1}}
	if !equalFairness(s.Fairness, want) {
		t.Errorf("fairness = %+v want %+v", s.Fairness, want)
	}
}

func TestBuild_InvalidDate(t *testing.T) {
	if _, err := Build(Request{Roster: []string{"A"}, Start: "2026-13-40", End: "2026-03-03"}); err == nil {
		t.Fatal("expected error for invalid start date")
	}
	if _, err := Build(Request{Roster: []string{"A"}, Start: "2026-03-02", End: "not-a-date"}); err == nil {
		t.Fatal("expected error for invalid end date")
	}
}

func TestBuild_EndBeforeStart(t *testing.T) {
	if _, err := Build(Request{Roster: []string{"A"}, Start: "2026-03-05", End: "2026-03-02"}); err == nil {
		t.Fatal("expected error for end before start")
	}
}

func TestBuild_StartIndexOutOfRange(t *testing.T) {
	if _, err := Build(Request{
		Roster: []string{"A"}, Start: "2026-03-02", End: "2026-03-03", StartIndex: 5,
	}); err == nil {
		t.Fatal("expected error for start index out of range")
	}
	if _, err := Build(Request{
		Roster: []string{"A"}, Start: "2026-03-02", End: "2026-03-03", StartIndex: -1,
	}); err == nil {
		t.Fatal("expected error for negative start index")
	}
}

func TestBuild_StartIndexIgnoredForEmptyRoster(t *testing.T) {
	// Empty roster ignores StartIndex (no range check) and stays uncovered.
	s, err := Build(Request{
		Roster: nil, Start: "2026-03-02", End: "2026-03-02", StartIndex: 99,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := entriesAsStrings(s); !equalStringSlice(got, []string{"uncovered"}) {
		t.Errorf("got %v want [uncovered]", got)
	}
}

func equalFairness(got, want []FairnessCount) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
