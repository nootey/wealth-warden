package jobs

import "time"

type dailyAt struct {
	hour   int
	minute int
}

func (s dailyAt) Next(current time.Time) time.Time {
	next := time.Date(current.Year(), current.Month(), current.Day(), s.hour, s.minute, 0, 0, current.Location())
	if !next.After(current) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
