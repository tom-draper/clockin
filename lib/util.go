package clockin

import "time"

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
