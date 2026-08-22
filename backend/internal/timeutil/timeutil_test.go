package timeutil

import (
	"strings"
	"testing"
	"time"
)

func TestFormatBeijing(t *testing.T) {
	ts := time.Date(2026, 8, 22, 23, 10, 3, 0, Beijing)
	got := Format(ts)
	if got != "2026-08-22 23:10:03" {
		t.Fatalf("got %s", got)
	}
	parsed, err := Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Hour() != 23 {
		t.Fatalf("hour=%d", parsed.Hour())
	}
}

func TestNowNotUTCStored(t *testing.T) {
	n := NowNaive()
	s := FormatNaive(n)
	if s == "" || !strings.Contains(s, "-") {
		t.Fatalf("empty format %q", s)
	}
}
