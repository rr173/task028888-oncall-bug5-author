package oncall

import "testing"

func TestBug5_RejectsInvalidBlackoutDate(t *testing.T) {
	_, err := Build(Request{
		Roster: []string{"alice"},
		Start:  "2026-03-02",
		End:    "2026-03-02",
		Blackouts: map[string]map[string]bool{
			"alice": {"not-a-date": true},
		},
	})
	if err == nil {
		t.Fatal("expected invalid blackout date to be rejected")
	}
}
