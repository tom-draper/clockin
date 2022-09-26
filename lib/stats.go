package clockin

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
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

func (a *All) fetchSessions(db *sql.DB) {
	sessions, err := getSessions(db, "")
	Check(err)
	a.sessions = sessions
}

func (t *Today) fetchSessions(db *sql.DB) {
	sessions, err := getSessions(db, "start BETWEEN CONCAT(CURDATE(), ' 00:00:00') AND CONCAT(CURDATE(), ' 23:59:59')")
	Check(err)
	t.sessions = sessions
}

func (d *Day) fetchSessions(db *sql.DB) {
	sessions, err := getSessions(db, "start >= DATE_SUB(NOW(), INTERVAL 1 DAY)")
	Check(err)
	d.sessions = sessions
}

func (w *Week) fetchSessions(db *sql.DB) {
	sessions, err := getSessions(db, "start >= DATE_SUB(NOW(), INTERVAL 1 WEEK)")
	Check(err)
	w.sessions = sessions
}

func (m *Month) fetchSessions(db *sql.DB) {
	sessions, err := getSessions(db, "start >= DATE_SUB(NOW(), INTERVAL 1 MONTH)")
	Check(err)
	m.sessions = sessions
}

func (y *Year) fetchSessions(db *sql.DB) {
	sessions, err := getSessions(db, "start >= DATE_SUB(NOW(), INTERVAL 1 YEAR)")
	Check(err)
	y.sessions = sessions
}

func basicInfo(sessions []Session) (*widgets.Paragraph, *widgets.Paragraph) {
	p := widgets.NewParagraph()
	p.TextStyle = ui.NewStyle(ui.ColorGreen)
	p.Title = "Sessions"
	p.Text = fmt.Sprintf("%d", len(sessions))
	p.PaddingLeft = 2
	p.SetRect(0, 4, 15, 7)

	p2 := widgets.NewParagraph()
	duration := totalDuration(sessions)
	p2.TextStyle = ui.NewStyle(ui.ColorGreen)
	p2.Title = "Total duration"
	p2.Text = durafmt.Parse(duration).LimitFirstN(3).String()
	p2.PaddingLeft = 2
	p2.SetRect(15, 4, 61, 7)

	return p, p2
}

func nameProportions(sessions []Session) (*widgets.PieChart, []ui.Drawable) {
	nameTime := make(map[string]float64)
	for _, session := range sessions {
		if session.Name != "" && !session.Finish.IsZero() {
			if _, ok := nameTime[session.Name]; !ok {
				nameTime[session.Name] = 0.0
			}
			nameTime[session.Name] += session.Finish.Sub(session.Start).Minutes()
		}
	}

	colors := []ui.Color{ui.ColorRed, ui.ColorGreen, ui.ColorYellow, ui.ColorBlue, ui.ColorCyan, ui.ColorMagenta}

	labels := []string{}
	data := []float64{}
	i := 0
	numColors := len(colors)
	for name, time := range nameTime {
		if time < 0 {
			time *= -1
		}
		if i > 5 {
			data[numColors] += time
		} else {
			data = append(data, time)
			labels = append(labels, name)
		}
		i++
	}
	if len(nameTime) > numColors {
		labels[numColors] = "Other"
	}

	pc := widgets.NewPieChart()
	pc.Title = "Session names"
	pc.Data = data
	pc.AngleOffset = -.5 * math.Pi
	pc.PaddingLeft = 1
	pc.PaddingTop = 2
	pc.PaddingRight = 1
	pc.PaddingBottom = 2
	pc.Colors = colors
	pc.LabelFormatter = func(i int, v float64) string {
		return fmt.Sprintf("%.02f", v)
	}
	pc.SetRect(61, 4, 112, 29)

	labelComponents := make([]ui.Drawable, len(labels))
	for i, name := range labels {
		p := widgets.NewParagraph()
		p.TextStyle = ui.NewStyle(ui.ColorGreen)
		p.Border = false
		if name == "" {
			name = "none"
		}
		p.Text = name
		p.TextStyle = ui.NewStyle(colors[i])
		p.PaddingLeft = 1
		p.SetRect(112, 4+(i*2), 140, 7+(i*2))
		labelComponents[i] = p
	}

	return pc, labelComponents
}

func (a *All) buildComponents() {
	p, p2 := basicInfo(a.sessions)

	bc := widgets.NewBarChart()
	data := make([]float64, 7)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0,
		now.Location())
	for _, session := range a.sessions {
		day := time.Date(session.Start.Year(),
			session.Start.Month(),
			session.Start.Day(), 0, 0, 0, 0,
			session.Start.Location())
		daysAgo := int(today.Sub(day).Hours() / 24.0)
		sessionDuration := session.Finish.Sub(session.Start).Minutes()
		data[6-(daysAgo%7)] += sessionDuration
	}

	for i, val := range data {
		data[i] = roundFloat(val, 0)
	}

	bc.Data = data
	bc.Labels = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	bc.Title = "Week average"
	bc.BarWidth = 7
	bc.PaddingLeft = 10
	bc.BarColors = []ui.Color{ui.ColorGreen}
	bc.LabelStyles = []ui.Style{ui.NewStyle(ui.ColorBlue)}
	bc.NumStyles = []ui.Style{ui.NewStyle(ui.ColorYellow)}
	bc.PaddingTop = 1
	bc.PaddingLeft = 2
	bc.PaddingRight = 2
	bc.SetRect(0, 7, 61, 29)

	pc, pcLabels := nameProportions(a.sessions)

	components := []ui.Drawable{p, p2, bc, pc}
	components = append(components, pcLabels...)
	a.components = components
}

func (t *Today) buildComponents() {
	p, p2 := basicInfo(t.sessions)

	nameTime := make(map[string]float64)
	for _, session := range t.sessions {
		if session.Name != "" && !session.Finish.IsZero() {
			if _, ok := nameTime[session.Name]; !ok {
				nameTime[session.Name] = 0.0
			}
			fmt.Println(session.Name, session.Finish.Sub(session.Start).Minutes())
			nameTime[session.Name] += session.Finish.Sub(session.Start).Minutes()
		}
	}

	pc, pcLabels := nameProportions(t.sessions)

	components := []ui.Drawable{p, p2, pc}
	components = append(components, pcLabels...)
	t.components = components
}

func (d *Day) buildComponents() {
	p, p2 := basicInfo(d.sessions)

	nameTime := make(map[string]float64)
	for _, session := range d.sessions {
		if session.Name != "" && !session.Finish.IsZero() {
			if _, ok := nameTime[session.Name]; !ok {
				nameTime[session.Name] = 0.0
			}
			fmt.Println(session.Name, session.Finish.Sub(session.Start).Minutes())
			nameTime[session.Name] += session.Finish.Sub(session.Start).Minutes()
		}
	}

	pc, pcLabels := nameProportions(d.sessions)

	components := []ui.Drawable{p, p2, pc}
	components = append(components, pcLabels...)
	d.components = components
}

func (w *Week) buildComponents() {
	p, p2 := basicInfo(w.sessions)

	bc := widgets.NewBarChart()
	data := make([]float64, 7)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0,
		now.Location())
	for _, session := range w.sessions {
		day := time.Date(session.Start.Year(),
			session.Start.Month(),
			session.Start.Day(), 0, 0, 0, 0,
			session.Start.Location())
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
	bc.BarWidth = 7
	bc.PaddingLeft = 10
	bc.BarColors = []ui.Color{ui.ColorGreen}
	bc.LabelStyles = []ui.Style{ui.NewStyle(ui.ColorBlue)}
	bc.NumStyles = []ui.Style{ui.NewStyle(ui.ColorYellow)}
	bc.PaddingTop = 1
	bc.PaddingLeft = 2
	bc.PaddingRight = 2
	bc.SetRect(0, 7, 61, 29)

	nameTime := make(map[string]float64)
	for _, session := range w.sessions {
		if session.Name != "" && !session.Finish.IsZero() {
			if _, ok := nameTime[session.Name]; !ok {
				nameTime[session.Name] = 0.0
			}
			fmt.Println(session.Name, session.Finish.Sub(session.Start).Minutes())
			nameTime[session.Name] += session.Finish.Sub(session.Start).Minutes()
		}
	}

	pc, pcLabels := nameProportions(w.sessions)

	components := []ui.Drawable{p, p2, bc, pc}
	components = append(components, pcLabels...)
	w.components = components
}

func (m *Month) buildComponents() {
	p, p2 := basicInfo(m.sessions)

	bc := widgets.NewBarChart()
	data := make([]float64, 7)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0,
		now.Location())
	for _, session := range m.sessions {
		day := time.Date(session.Start.Year(),
			session.Start.Month(),
			session.Start.Day(), 0, 0, 0, 0,
			session.Start.Location())
		daysAgo := int(today.Sub(day).Hours() / 24.0)
		sessionDuration := session.Finish.Sub(session.Start).Minutes()
		data[6-(daysAgo%7)] += sessionDuration
	}

	for i, val := range data {
		data[i] = roundFloat(val, 0)
	}

	bc.Data = data
	bc.Labels = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	bc.Title = "Week average"
	bc.BarWidth = 7
	bc.PaddingLeft = 10
	bc.BarColors = []ui.Color{ui.ColorGreen}
	bc.LabelStyles = []ui.Style{ui.NewStyle(ui.ColorBlue)}
	bc.NumStyles = []ui.Style{ui.NewStyle(ui.ColorYellow)}
	bc.PaddingTop = 1
	bc.PaddingLeft = 2
	bc.PaddingRight = 2
	bc.SetRect(0, 7, 61, 29)

	nameTime := make(map[string]float64)
	for _, session := range m.sessions {
		if session.Name != "" && !session.Finish.IsZero() {
			if _, ok := nameTime[session.Name]; !ok {
				nameTime[session.Name] = 0.0
			}
			fmt.Println(session.Name, session.Finish.Sub(session.Start).Minutes())
			nameTime[session.Name] += session.Finish.Sub(session.Start).Minutes()
		}
	}

	pc, pcLabels := nameProportions(m.sessions)

	components := []ui.Drawable{p, p2, bc, pc}
	components = append(components, pcLabels...)
	m.components = components
}

func (y *Year) buildComponents() {
	p, p2 := basicInfo(y.sessions)

	bc := widgets.NewBarChart()
	data := make([]float64, 7)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0,
		now.Location())
	for _, session := range y.sessions {
		day := time.Date(session.Start.Year(),
			session.Start.Month(),
			session.Start.Day(), 0, 0, 0, 0,
			session.Start.Location())
		daysAgo := int(today.Sub(day).Hours() / 24.0)
		sessionDuration := session.Finish.Sub(session.Start).Minutes()
		data[6-(daysAgo%7)] += sessionDuration
	}

	for i, val := range data {
		data[i] = roundFloat(val, 0)
	}

	bc.Data = data
	bc.Labels = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	bc.Title = "Week average"
	bc.BarWidth = 7
	bc.PaddingLeft = 10
	bc.BarColors = []ui.Color{ui.ColorGreen}
	bc.LabelStyles = []ui.Style{ui.NewStyle(ui.ColorBlue)}
	bc.NumStyles = []ui.Style{ui.NewStyle(ui.ColorYellow)}
	bc.PaddingTop = 1
	bc.PaddingLeft = 2
	bc.PaddingRight = 2
	bc.SetRect(0, 7, 61, 29)

	nameTime := make(map[string]float64)
	for _, session := range y.sessions {
		if session.Name != "" && !session.Finish.IsZero() {
			if _, ok := nameTime[session.Name]; !ok {
				nameTime[session.Name] = 0.0
			}
			fmt.Println(session.Name, session.Finish.Sub(session.Start).Minutes())
			nameTime[session.Name] += session.Finish.Sub(session.Start).Minutes()
		}
	}

	pc, pcLabels := nameProportions(y.sessions)

	components := []ui.Drawable{p, p2, bc, pc}
	components = append(components, pcLabels...)
	y.components = components
}

func (a All) render() {
	for _, component := range a.components {
		ui.Render(component)
	}
}

func (t Today) render() {
	for _, component := range t.components {
		ui.Render(component)
	}
}

func (d Day) render() {
	for _, component := range d.components {
		ui.Render(component)
	}
}

func (w Week) render() {
	for _, component := range w.components {
		ui.Render(component)
	}
}

func (m Month) render() {
	for _, component := range m.components {
		ui.Render(component)
	}
}

func (y Year) render() {
	for _, component := range y.components {
		ui.Render(component)
	}
}

type Period interface {
	fetchSessions(db *sql.DB)
	buildComponents()
	render()
}

type All struct {
	sessions   []Session
	components []ui.Drawable
}

type Today struct {
	sessions   []Session
	components []ui.Drawable
}

type Day struct {
	sessions   []Session
	components []ui.Drawable
}

type Week struct {
	sessions   []Session
	components []ui.Drawable
}

type Month struct {
	sessions   []Session
	components []ui.Drawable
}

type Year struct {
	sessions   []Session
	components []ui.Drawable
}

func buildPages(db *sql.DB) []Period {
	all := All{}
	today := Today{}
	day := Day{}
	week := Week{}
	month := Month{}
	year := Year{}
	pages := []Period{&all, &today, &day, &week, &month, &year}

	for _, page := range pages {
		page.fetchSessions(db)
		page.buildComponents()
	}
	return pages
}

func DisplayStats(db *sql.DB, period string) error {
	pages := buildPages(db)

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	tabpane := widgets.NewTabPane("All", "Today", "24hrs", "Week", "Month", "Year")
	tabpane.SetRect(0, 1, 50, 3)
	tabpane.Border = false

	signOff := widgets.NewParagraph()
	signOff.Border = false
	signOff.Text = "Use arrow keys to switch tabs.\nPress esc to quit."
	signOff.SetRect(0, 29, 58, 34)
	ui.Render(signOff)

	renderTab := func() {
		page := pages[tabpane.ActiveTabIndex]
		page.render()
	}

	ui.Render(tabpane, signOff)
	renderTab()

	uiEvents := ui.PollEvents()

	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>", "<Escape>":
			return nil
		case "<Left>", "p", "l":
			tabpane.FocusLeft()
			ui.Clear()
			ui.Render(tabpane, signOff)
			renderTab()
		case "<Right>", "n", "r":
			tabpane.FocusRight()
			ui.Clear()
			ui.Render(tabpane, signOff)
			renderTab()
		}
	}
}
