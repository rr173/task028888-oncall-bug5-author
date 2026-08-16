// Command oncall generates an on-call rotation schedule over a date range.
//
// Usage:
//
//	oncall --roster alice,bob,carol --start 2026-03-02 --end 2026-03-08 \
//	       --start-index 0 --holidays 2026-03-07 --blackouts alice=2026-03-03
//	oncall --smoke-test
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"oncall/internal/oncall"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("oncall", flag.ContinueOnError)
	var (
		rosterStr  string
		start      string
		end        string
		startIndex int
		holidays   string
		blackouts  string
		smoke      bool
	)
	fs.StringVar(&rosterStr, "roster", "", "comma-separated engineer IDs")
	fs.StringVar(&start, "start", "", "start date YYYY-MM-DD (inclusive)")
	fs.StringVar(&end, "end", "", "end date YYYY-MM-DD (inclusive)")
	fs.IntVar(&startIndex, "start-index", 0, "0-based roster index for the first business day")
	fs.StringVar(&holidays, "holidays", "", "comma-separated holiday dates YYYY-MM-DD")
	fs.StringVar(&blackouts, "blackouts", "", "comma-separated eng=date entries, e.g. alice=2026-03-02")
	fs.BoolVar(&smoke, "smoke-test", false, "run built-in smoke test and exit")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if smoke {
		return runSmokeTest()
	}

	if start == "" || end == "" {
		return fmt.Errorf("--start and --end are required (or use --smoke-test)")
	}

	req := oncall.Request{
		Roster:     splitCSV(rosterStr),
		Start:      start,
		End:        end,
		StartIndex: startIndex,
		Holidays:   setFromCSV(holidays),
		Blackouts:  parseBlackouts(blackouts),
	}

	sched, err := oncall.Build(req)
	if err != nil {
		return err
	}
	printSchedule(sched)
	return nil
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func setFromCSV(s string) map[string]bool {
	parts := splitCSV(s)
	if len(parts) == 0 {
		return nil
	}
	m := make(map[string]bool, len(parts))
	for _, p := range parts {
		m[p] = true
	}
	return m
}

func parseBlackouts(s string) map[string]map[string]bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	out := make(map[string]map[string]bool)
	for _, entry := range strings.Split(s, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		kv := strings.SplitN(entry, "=", 2)
		if len(kv) != 2 {
			continue
		}
		eng := strings.TrimSpace(kv[0])
		date := strings.TrimSpace(kv[1])
		if eng == "" || date == "" {
			continue
		}
		if _, ok := out[eng]; !ok {
			out[eng] = make(map[string]bool)
		}
		out[eng][date] = true
	}
	return out
}

func printSchedule(s *oncall.Schedule) {
	fmt.Printf("%-12s %-10s %-12s %s\n", "DATE", "WEEKDAY", "STATUS", "ENGINEER")
	for _, e := range s.Entries {
		fmt.Printf("%-12s %-10s %-12s %s\n", e.Date, e.Weekday, e.Status, e.Engineer)
	}
	fmt.Println("\nFAIRNESS:")
	for _, f := range s.Fairness {
		fmt.Printf("  %s: %d\n", f.Engineer, f.Days)
	}
}

func runSmokeTest() error {
	type scenario struct {
		name string
		req  oncall.Request
		want []string
	}
	cases := []scenario{
		{
			name: "plain round-robin",
			req: oncall.Request{
				Roster:     []string{"A", "B", "C"},
				Start:      "2026-03-02",
				End:        "2026-03-07",
				StartIndex: 0,
			},
			want: []string{"A", "B", "C", "A", "B", "C"},
		},
		{
			name: "holiday pause",
			req: oncall.Request{
				Roster:     []string{"A", "B", "C"},
				Start:      "2026-03-02",
				End:        "2026-03-06",
				StartIndex: 0,
				Holidays:   map[string]bool{"2026-03-03": true},
			},
			want: []string{"A", "holiday", "B", "C", "A"},
		},
		{
			name: "blackout skip",
			req: oncall.Request{
				Roster:     []string{"A", "B", "C"},
				Start:      "2026-03-02",
				End:        "2026-03-04",
				StartIndex: 0,
				Blackouts:  map[string]map[string]bool{"A": {"2026-03-02": true}},
			},
			want: []string{"B", "C", "A"},
		},
		{
			name: "all blackout uncovered",
			req: oncall.Request{
				Roster:     []string{"A"},
				Start:      "2026-03-02",
				End:        "2026-03-03",
				StartIndex: 0,
				Blackouts:  map[string]map[string]bool{"A": {"2026-03-02": true}},
			},
			want: []string{"uncovered", "A"},
		},
		{
			name: "empty roster",
			req: oncall.Request{
				Roster: nil,
				Start:  "2026-03-02",
				End:    "2026-03-03",
			},
			want: []string{"uncovered", "uncovered"},
		},
	}

	ok := true
	for _, c := range cases {
		s, err := oncall.Build(c.req)
		if err != nil {
			fmt.Printf("[FAIL] %s: build error: %v\n", c.name, err)
			ok = false
			continue
		}
		got := make([]string, 0, len(s.Entries))
		for _, e := range s.Entries {
			if e.Status == oncall.StatusAssigned {
				got = append(got, e.Engineer)
			} else {
				got = append(got, string(e.Status))
			}
		}
		if !equalStrings(got, c.want) {
			fmt.Printf("[FAIL] %s: got %v want %v\n", c.name, got, c.want)
			ok = false
			continue
		}
		fmt.Printf("[OK]   %s: %v\n", c.name, got)
	}

	if _, err := oncall.Build(oncall.Request{
		Roster: []string{"A"}, Start: "2026-03-05", End: "2026-03-02",
	}); err == nil {
		fmt.Println("[FAIL] expected error for end before start")
		ok = false
	} else {
		fmt.Println("[OK]   end-before-start rejected")
	}
	if _, err := oncall.Build(oncall.Request{
		Roster: []string{"A"}, Start: "2026-03-02", End: "2026-03-03", StartIndex: 9,
	}); err == nil {
		fmt.Println("[FAIL] expected error for start index out of range")
		ok = false
	} else {
		fmt.Println("[OK]   start-index out of range rejected")
	}

	if !ok {
		return fmt.Errorf("smoke test failed")
	}
	fmt.Println("\nsmoke test passed")
	return nil
}

func equalStrings(a, b []string) bool {
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
