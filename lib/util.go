package clockin

import (
	"time"

	"github.com/hako/durafmt"
)

func Check(err error) {
	if err != nil {
		panic(err)
	}
}

func CurrentTime() time.Time {
	// Parse time to force into local time
	now, err := time.Parse("2006-01-02 15:04:05", time.Now().Format("2006-01-02 15:04:05"))
	Check(err)
	return now
}

func formatDuration(duration time.Duration, limitFirstN int) string {
	return durafmt.Parse(duration).LimitFirstN(limitFirstN).String()
}
