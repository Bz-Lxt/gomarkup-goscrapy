package timeutil

import (
	"fmt"
	"strings"
	"time"
)

const (
	Layout     = "2006-01-02 15:04:05"
	DateLayout = "2006-01-02"
)

// Beijing is GMT+8. All persisted timestamps use this zone as naive local time.
var Beijing = time.FixedZone("CST", 8*60*60)

func Now() time.Time {
	return time.Now().In(Beijing)
}

// Naive returns the Beijing wall-clock time without a location, suitable for
// TIMESTAMP-without-timezone columns. Never persist UTC wall-clock values.
func Naive(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	loc := t.In(Beijing)
	return time.Date(loc.Year(), loc.Month(), loc.Day(), loc.Hour(), loc.Minute(), loc.Second(), loc.Nanosecond(), time.Local)
}

func NowNaive() time.Time {
	return Naive(Now())
}

func Format(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(Beijing).Format(Layout)
}

func FormatNaive(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	if t.Location() == time.UTC {
		// Treat stored naive values as Beijing wall-clock, not UTC.
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), Beijing).Format(Layout)
	}
	return t.In(Beijing).Format(Layout)
}

func Parse(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	if t, err := time.ParseInLocation(Layout, s, Beijing); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation(time.RFC3339, s, Beijing); err == nil {
		return t.In(Beijing), nil
	}
	return time.Time{}, fmt.Errorf("parse time %q", s)
}

func Unix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.In(Beijing).Unix()
}

func FromUnix(sec int64) time.Time {
	return time.Unix(sec, 0).In(Beijing)
}

func Since(t time.Time) time.Duration {
	return Now().Sub(t.In(Beijing))
}

func Add(t time.Time, d time.Duration) time.Time {
	return t.In(Beijing).Add(d)
}

func StartOfDay(t time.Time) time.Time {
	loc := t.In(Beijing)
	return time.Date(loc.Year(), loc.Month(), loc.Day(), 0, 0, 0, 0, Beijing)
}
