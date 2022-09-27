package clockin

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func calcDuration(session Session) time.Duration {
	return session.Finish.Sub(session.Start)
}

func totalDuration(sessions []Session) time.Duration {
	var totalDuration time.Duration
	for _, session := range sessions {
		if !session.Finish.IsZero() {
			duration := calcDuration(session)
			totalDuration += duration
		}
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

func getSessions(db *sql.DB, includeActive bool, sqlDateRange string) ([]Session, error) {
	query := "SELECT * FROM clockin"
	if !includeActive {
		query += " WHERE FINISH IS NOT NULL"
		if sqlDateRange != "" {
			query += " AND " + sqlDateRange
		}
	} else if sqlDateRange != "" {
		query += " WHERE " + sqlDateRange
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	sessions := ExtractSessions(rows)
	return sessions, nil
}

func (a *All) fetchSessions(db *sql.DB) {
	sessions, err := getSessions(db, true, "")
	Check(err)
	a.sessions = sessions
}

func (t *Today) fetchSessions(db *sql.DB) {
	sessions, err := getSessions(db, true, "start BETWEEN CONCAT(CURDATE(), ' 00:00:00') AND CONCAT(CURDATE(), ' 23:59:59')")
	Check(err)
	t.sessions = sessions
}

func (d *Day) fetchSessions(db *sql.DB) {
	sessions, err := getSessions(db, true, "start >= DATE_SUB(NOW(), INTERVAL 1 DAY)")
	Check(err)
	d.sessions = sessions
}

func (w *Week) fetchSessions(db *sql.DB) {
	sessions, err := getSessions(db, true, "start >= DATE_SUB(NOW(), INTERVAL 1 WEEK)")
	Check(err)
	w.sessions = sessions
}

func (m *Month) fetchSessions(db *sql.DB) {
	sessions, err := getSessions(db, true, "start >= DATE_SUB(NOW(), INTERVAL 1 MONTH)")
	Check(err)
	m.sessions = sessions
}

func (y *Year) fetchSessions(db *sql.DB) {
	sessions, err := getSessions(db, true, "start >= DATE_SUB(NOW(), INTERVAL 1 YEAR)")
	Check(err)
	y.sessions = sessions
}

func numActive(sessions []Session) int {
	count := 0
	for _, session := range sessions {
		if session.Finish.IsZero() {
			count++
		}
	}
	return count
}

func basicInfo(sessions []Session) (*widgets.Paragraph, *widgets.Paragraph,
	*widgets.Paragraph) {
	p := widgets.NewParagraph()
	duration := totalDuration(sessions)
	p.TextStyle = ui.NewStyle(ui.ColorGreen)
	p.Title = "Total duration"
	p.Text = formatDuration(duration, 3)
	p.PaddingLeft = 2
	p.SetRect(0, 4, 61, 7)

	p2 := widgets.NewParagraph()
	p2.TextStyle = ui.NewStyle(ui.ColorGreen)
	p2.Title = "Completed"
	p2.Text = fmt.Sprintf("%d", len(sessions))
	p2.PaddingLeft = 2
	p2.SetRect(0, 7, 30, 10)

	p3 := widgets.NewParagraph()
	p3.TextStyle = ui.NewStyle(ui.ColorYellow)
	p3.Title = "Active"
	p3.Text = fmt.Sprintf("%d", numActive(sessions))
	p3.PaddingLeft = 2
	p3.SetRect(30, 7, 61, 10)

	return p, p2, p3
}

type PieChartData struct {
	data   []float64
	labels []string
}

type SortByOther PieChartData

func (sbo SortByOther) Len() int {
	return len(sbo.data)
}

func (sbo SortByOther) Swap(i, j int) {
	sbo.data[i], sbo.data[j] = sbo.data[j], sbo.data[i]
	sbo.labels[i], sbo.labels[j] = sbo.labels[j], sbo.labels[i]
}

func (sbo SortByOther) Less(i, j int) bool {
	return sbo.data[i] > sbo.data[j]
}

func nameProportions(sessions []Session) (*widgets.PieChart, []ui.Drawable) {
	nameTime := make(map[string]float64)
	for _, session := range sessions {
		if !session.Finish.IsZero() {
			if _, ok := nameTime[session.Name]; !ok {
				nameTime[session.Name] = 0.0
			}
			nameTime[session.Name] += session.Finish.Sub(session.Start).Minutes()
		}
	}

	colors := []ui.Color{ui.ColorRed, ui.ColorGreen, ui.ColorYellow,
		ui.ColorBlue, ui.ColorCyan, ui.ColorMagenta}

	labels := []string{}
	data := []float64{}
	i := 0
	numColors := len(colors)
	for name, time := range nameTime {
		if time < 0 {
			time *= -1
		}
		if i > 5 {
			data[numColors-1] += time
		} else {
			data = append(data, time)
			labels = append(labels, name)
		}
		i++
	}
	if len(nameTime) > numColors {
		labels[numColors-1] = "Other"
	}

	pcData := PieChartData{data: data, labels: labels}
	sort.Sort(SortByOther(pcData))

	pc := widgets.NewPieChart()
	pc.Title = "Session names"
	pc.Data = pcData.data
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

	labelComponents := make([]ui.Drawable, len(pcData.labels))
	for i, name := range pcData.labels {
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

func sessionsList(sessions []Session) *widgets.List {
	l := widgets.NewList()
	l.Title = "Sessions"
	rows := []string{}
	for _, session := range sessions {
		name := "none"
		if session.Name != "" {
			name = session.Name
		}
		duration := totalDuration(sessions)
		rows = append(rows, fmt.Sprintf("[%d] %s - %s", session.ID, name,
			formatDuration(duration, 3)))
	}
	l.Rows = rows
	l.PaddingLeft = 2
	l.PaddingRight = 2
	l.PaddingTop = 1
	l.PaddingBottom = 1
	l.WrapText = true
	// l.TextStyle = ui.NewStyle(ui.ColorYellow)
	l.SetRect(0, 10, 61, 29)
	return l
}

func weekAverage(sessions []Session) *widgets.BarChart {
	bc := widgets.NewBarChart()
	data := make([]float64, 7)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0,
		now.Location())
	for _, session := range sessions {
		if !session.Finish.IsZero() {
			day := time.Date(session.Start.Year(),
				session.Start.Month(),
				session.Start.Day(), 0, 0, 0, 0,
				session.Start.Location())
			daysAgo := int(today.Sub(day).Hours() / 24.0)
			sessionDuration := session.Finish.Sub(session.Start).Minutes()
			data[6-daysAgo] += sessionDuration
		}
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
	bc.SetRect(0, 10, 61, 29)

	return bc
}

func (a *All) buildComponents() {
	p, p2, p3 := basicInfo(a.sessions)

	bc := weekAverage(a.sessions)

	pc, pcLabels := nameProportions(a.sessions)

	components := []ui.Drawable{p, p2, p3, bc, pc}
	components = append(components, pcLabels...)
	a.components = components
}

func (t *Today) buildComponents() {
	p, p2, p3 := basicInfo(t.sessions)

	nameTime := make(map[string]float64)
	for _, session := range t.sessions {
		if session.Name != "" && !session.Finish.IsZero() {
			if _, ok := nameTime[session.Name]; !ok {
				nameTime[session.Name] = 0.0
			}
			nameTime[session.Name] += session.Finish.Sub(session.Start).Minutes()
		}
	}

	l := sessionsList(t.sessions)

	pc, pcLabels := nameProportions(t.sessions)

	components := []ui.Drawable{p, p2, p3, pc, l}
	components = append(components, pcLabels...)
	t.components = components
	t.list = l
}

func (d *Day) buildComponents() {
	p, p2, p3 := basicInfo(d.sessions)

	nameTime := make(map[string]float64)
	for _, session := range d.sessions {
		if session.Name != "" && !session.Finish.IsZero() {
			if _, ok := nameTime[session.Name]; !ok {
				nameTime[session.Name] = 0.0
			}
			nameTime[session.Name] += session.Finish.Sub(session.Start).Minutes()
		}
	}

	l := sessionsList(d.sessions)

	pc, pcLabels := nameProportions(d.sessions)

	components := []ui.Drawable{p, p2, p3, pc, l}
	components = append(components, pcLabels...)
	d.components = components
	d.list = l
}

func (w *Week) buildComponents() {
	p, p2, p3 := basicInfo(w.sessions)

	bc := weekAverage(w.sessions)

	nameTime := make(map[string]float64)
	for _, session := range w.sessions {
		if session.Name != "" && !session.Finish.IsZero() {
			if _, ok := nameTime[session.Name]; !ok {
				nameTime[session.Name] = 0.0
			}
			nameTime[session.Name] += session.Finish.Sub(session.Start).Minutes()
		}
	}

	pc, pcLabels := nameProportions(w.sessions)

	components := []ui.Drawable{p, p2, p3, bc, pc}
	components = append(components, pcLabels...)
	w.components = components
}

func (m *Month) buildComponents() {
	p, p2, p3 := basicInfo(m.sessions)

	bc := weekAverage(m.sessions)

	nameTime := make(map[string]float64)
	for _, session := range m.sessions {
		if session.Name != "" && !session.Finish.IsZero() {
			if _, ok := nameTime[session.Name]; !ok {
				nameTime[session.Name] = 0.0
			}
			nameTime[session.Name] += session.Finish.Sub(session.Start).Minutes()
		}
	}

	pc, pcLabels := nameProportions(m.sessions)

	components := []ui.Drawable{p, p2, p3, bc, pc}
	components = append(components, pcLabels...)
	m.components = components
}

func (y *Year) buildComponents() {
	p, p2, p3 := basicInfo(y.sessions)

	bc := weekAverage(y.sessions)

	nameTime := make(map[string]float64)
	for _, session := range y.sessions {
		if session.Name != "" && !session.Finish.IsZero() {
			if _, ok := nameTime[session.Name]; !ok {
				nameTime[session.Name] = 0.0
			}
			nameTime[session.Name] += session.Finish.Sub(session.Start).Minutes()
		}
	}

	pc, pcLabels := nameProportions(y.sessions)

	components := []ui.Drawable{p, p2, p3, bc, pc}
	components = append(components, pcLabels...)
	y.components = components
}

func (a All) scroll(direction string) {
	if a.list != nil {
		if direction == "up" {
			a.list.ScrollUp()
			ui.Render(a.list)
		} else if direction == "down" {
			a.list.ScrollDown()
			ui.Render(a.list)
		}
	}
}

func (t Today) scroll(direction string) {
	if t.list != nil {
		if direction == "up" {
			t.list.ScrollPageUp()
			ui.Render(t.list)
		} else if direction == "down" {
			t.list.ScrollPageDown()
			ui.Render(t.list)
		}
	}
}

func (d Day) scroll(direction string) {
	if d.list != nil {
		if direction == "up" {
			d.list.ScrollPageUp()
			ui.Render(d.list)
		} else if direction == "down" {
			d.list.ScrollPageDown()
			ui.Render(d.list)
		}
	}
}

func (w Week) scroll(direction string) {
	if w.list != nil {
		if direction == "up" {
			w.list.ScrollPageUp()
			ui.Render(w.list)
		} else if direction == "down" {
			w.list.ScrollPageDown()
			ui.Render(w.list)
		}
	}
}

func (m Month) scroll(direction string) {
	if m.list != nil {
		if direction == "up" {
			m.list.ScrollPageUp()
			ui.Render(m.list)
		} else if direction == "down" {
			m.list.ScrollPageDown()
			ui.Render(m.list)
		}
	}
}

func (y Year) scroll(direction string) {
	if y.list != nil {
		if direction == "up" {
			y.list.ScrollPageUp()
			ui.Render(y.list)
		} else if direction == "down" {
			y.list.ScrollPageDown()
			ui.Render(y.list)
		}
	}
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

type Page interface {
	fetchSessions(db *sql.DB)
	buildComponents()
	scroll(direction string)
	render()
}

type All struct {
	sessions   []Session
	components []ui.Drawable
	list       *widgets.List
}

type Today struct {
	sessions   []Session
	components []ui.Drawable
	list       *widgets.List
}

type Day struct {
	sessions   []Session
	components []ui.Drawable
	list       *widgets.List
}

type Week struct {
	sessions   []Session
	components []ui.Drawable
	list       *widgets.List
}

type Month struct {
	sessions   []Session
	components []ui.Drawable
	list       *widgets.List
}

type Year struct {
	sessions   []Session
	components []ui.Drawable
	list       *widgets.List
}

func buildPages(db *sql.DB) []Page {
	all := All{}
	today := Today{}
	day := Day{}
	week := Week{}
	month := Month{}
	year := Year{}
	pages := []Page{&all, &today, &day, &week, &month, &year}

	for _, page := range pages {
		page.fetchSessions(db)
		page.buildComponents()
	}
	return pages
}

func DisplayStats(db *sql.DB) error {
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
		case "j", "<Down>":
			if tabpane.ActiveTabIndex == 1 || tabpane.ActiveTabIndex == 2 {
				pages[tabpane.ActiveTabIndex].scroll("down")
			}
		case "k", "<Up>":
			if tabpane.ActiveTabIndex == 1 || tabpane.ActiveTabIndex == 2 {
				pages[tabpane.ActiveTabIndex].scroll("up")
			}
		}
	}
}
