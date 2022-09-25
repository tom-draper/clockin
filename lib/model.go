package clockin

import "time"

type Session struct {
	ID     int
	Name   string
	Start  time.Time
	Finish time.Time
}
