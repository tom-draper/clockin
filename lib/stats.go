package clockin

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/guptarohit/asciigraph"
	"github.com/hako/durafmt"
)

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func totalDuration(sessions []Session) time.Duration {
	var totalDuration time.Duration
	for _, session := range sessions {
		duration := session.Finish.Sub(session.Start)
		totalDuration += duration
	}
	return totalDuration
}

func displayDurationString(duration time.Duration, time string) string {
	var str string
	switch time {
	case "":
		str = "Total duration: "
	case "today":
		str = "Total duration today: "
	case "day":
		str = "Total duration in last 24 hours: "
	case "week":
		str = "Total duration in last week: "
	case "month":
		str = "Total duration in last month: "
	case "year":
		str = "Total duration in last year: "
	}
	str += durafmt.Parse(duration).LimitFirstN(2).String()
	return str
}

func ExtractSessions(rows *sql.Rows) []Session {
	var sessions []Session
	for rows.Next() {
		var session Session
		rows.Scan(&session.ID, &session.Name, &session.Start, &session.Finish)
		sessions = append(sessions, session)
	}
	return sessions
}

func getSessions(db *sql.DB, sqlDateRange string) ([]Session, error) {
	var rows *sql.Rows
	var err error
	if sqlDateRange == "" {
		rows, err = db.Query("SELECT * FROM clockin WHERE FINISH IS NOT NULL")
	} else {
		rows, err = db.Query("SELECT * FROM clockin WHERE FINISH IS NOT NULL AND " + sqlDateRange)
	}
	if err != nil {
		return nil, err
	}

	sessions := ExtractSessions(rows)
	return sessions, nil
}

func getAllSessions(db *sql.DB) ([]Session, error) {
	return getSessions(db, "")
}

func getSessionsToday(db *sql.DB) ([]Session, error) {
	return getSessions(db, "start BETWEEN NOW() AND CURRENT_DATE() + INTERVAL 1 DAY)")
}

func getSessionsDay(db *sql.DB) ([]Session, error) {
	return getSessions(db, "start >= DATE_SUB(NOW(), INTERVAL 1 DAY)")
}

func getSessionsWeek(db *sql.DB) ([]Session, error) {
	return getSessions(db, "start >= DATE_SUB(NOW(), INTERVAL 1 WEEK)")
}

func getSessionsMonth(db *sql.DB) ([]Session, error) {
	return getSessions(db, "start >= DATE_SUB(NOW(), INTERVAL 1 MONTH)")
}

func getSessionsYear(db *sql.DB) ([]Session, error) {
	return getSessions(db, "start >= DATE_SUB(NOW(), INTERVAL 1 YEAR)")
}

func DisplayStats(db *sql.DB, period string) error {
	var sessions []Session
	var err error
	switch period {
	case "":
		fmt.Println("Statistics:")
		sessions, err = getAllSessions(db)
	case "today":
		fmt.Println("Sessions from today:")
		sessions, err = getSessionsToday(db)
	case "day":
		fmt.Println("Sessions from last 24hrs:")
		sessions, err = getSessionsDay(db)
	case "week":
		fmt.Println("Sessions from last week:")
		sessions, err = getSessionsWeek(db)
	case "month":
		fmt.Println("Sessions from last month:")
		sessions, err = getSessionsMonth(db)
	case "year":
		fmt.Println("Sessions from last year:")
		sessions, err = getSessionsYear(db)

	}
	if err != nil {
		log.Printf("Sessions in time range failed with error: %s\n", err)
		return err
	}

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	p := widgets.NewParagraph()
	duration := totalDuration(sessions)
	durationStr := displayDurationString(duration, period)
	p.Border = false
	p.TextStyle = ui.NewStyle(ui.ColorGreen)
	p.Text = fmt.Sprintf("%d sessions\n%s", len(sessions), durationStr)
	p.SetRect(0, 0, 57, 4)
	ui.Render(p)

	if period == "week" {
		bc := widgets.NewBarChart()
		data := []float64{100.1, 100.1, 100.2, 200.44, 100.1, 100.3, 100.4}
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		for _, session := range sessions {
			day := time.Date(session.Start.Year(), session.Start.Month(), session.Start.Day(), 0, 0, 0, 0, session.Start.Location())
			daysAgo := int(today.Sub(day).Hours() / 24.0)
			sessionDuration := session.Finish.Sub(session.Start).Minutes()
			data[6-daysAgo] += sessionDuration
		}

		for i, val := range data {
			data[i] = roundFloat(val, 0)
		}

		bc.Data = data
		bc.Labels = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
		bc.Title = "Last 7 days"
		bc.SetRect(0, 4, 57, 25)
		bc.BarWidth = 7
		bc.PaddingLeft = 10
		bc.BarColors = []ui.Color{ui.ColorGreen}
		bc.LabelStyles = []ui.Style{ui.NewStyle(ui.ColorBlue)}
		bc.NumStyles = []ui.Style{ui.NewStyle(ui.ColorYellow)}

		ui.Render(bc)
	} else if period == "month" || period == "year" {
		var nDays int
		if period == "month" {
			nDays = 30
		} else if period == "year" {
			nDays = 365
		}
		data := make([]float64, nDays)
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		for _, session := range sessions {
			day := time.Date(session.Start.Year(), session.Start.Month(), session.Start.Day(), 0, 0, 0, 0, session.Start.Location())
			daysAgo := int(today.Sub(day).Hours() / 24.0)
			sessionDuration := session.Finish.Sub(session.Start).Minutes()
			data[nDays-1-daysAgo] += sessionDuration
		}
		graph := asciigraph.Plot(data, asciigraph.Width(60))

		fmt.Printf("\n%s\n\n", graph)
	}

	p = widgets.NewParagraph()
	p.Border = false
	p.Text = "Press any key to exit"
	p.SetRect(0, 25, 57, 28)
	ui.Render(p)

	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			break
		}
	}

	return nil
}
