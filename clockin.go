package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/TwiN/go-color"
	_ "github.com/go-sql-driver/mysql"
	"github.com/guptarohit/asciigraph"
	"github.com/hako/durafmt"
	"github.com/joho/godotenv"
)

const (
	hostname = "127.0.0.1:3306"
	dbname   = "clockin"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func dsn(username string, password string, dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", username, password, hostname, dbName)
}

func rowsAffected(res sql.Result) (int64, error) {
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error when getting rows affected: %s\n", err)
		return 0, err
	}
	return rows, nil
}

func dbConnection(username string, password string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn(username, password, ""))
	if err != nil {
		log.Printf("Error when opening database: %s\n", err)
		return nil, err
	}

	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	_, err = db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+dbname)
	if err != nil {
		log.Printf("Error when creating database: %s\n", err)
		return nil, err
	}

	db.Close()
	db, err = sql.Open("mysql", dsn(username, password, dbname))
	if err != nil {
		log.Printf("Error when opening database: %s\n", err)
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Minute * 5)
	return db, nil
}

func createTable(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS clockin(id int primary key auto_increment, name varchar(100), start datetime default CURRENT_TIMESTAMP, finish datetime)`
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	_, err := db.ExecContext(ctx, query)
	if err != nil {
		log.Printf("Error when creating table: %s\n", err)
		return err
	}

	return nil
}

func showTable(db *sql.DB) error {
	res, err := db.Query("SELECT * FROM clockin")
	if err != nil {
		return err
	}

	for res.Next() {
		var session Session
		res.Scan(&session.id, &session.name, &session.start, &session.finish)
		name := session.name
		if session.name == "" {
			name = "none"
		}
		if session.finish.IsZero() {
			fmt.Printf("%d %s %s %s\n", session.id, name, session.start, color.Ize(color.Yellow, session.finish.String()))
		} else {
			fmt.Printf("%d %s %s %s\n", session.id, name, session.start, session.finish)
		}
	}

	return nil
}

func startRecording(db *sql.DB, name string) error {
	query := "INSERT INTO clockin(name, start, finish) VALUES (?, ?, NULL)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error when preparing SQL insert statement: %s\n", err)
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC()
	_, err = stmt.ExecContext(ctx, name, now)
	if err != nil {
		log.Printf("Error when inserting row into products table: %s\n", err)
		return err
	}

	if name == "" {
		fmt.Printf(color.Ize(color.Green, "Started recording (%s)\n"), now)
	} else {
		fmt.Printf(color.Ize(color.Green, "Started recording %s (%s)\n"), name, now)
	}
	return nil
}

func finishRecording(db *sql.DB, name string) error {
	var query string
	if name == "" {
		query = "UPDATE clockin set finish=NOW() WHERE finish is NULL"
	} else {
		query = "UPDATE clockin set finish=NOW() WHERE finish is NULL AND name=?"
	}

	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error when preparing SQL update statement: %s\n", err)
		return err
	}
	defer stmt.Close()

	var res sql.Result
	if name == "" || name == "all" {
		res, err = stmt.ExecContext(ctx)
	} else {
		res, err = stmt.ExecContext(ctx, name)
	}
	if err != nil {
		log.Printf("Error when inserting row into products table: %s\n", err)
		return err
	}

	n, err := rowsAffected(res)
	if err != nil {
		log.Printf("Error when finding rows affected: %s\n", err)
		return err
	}

	if name == "" {
		if n == 0 {
			fmt.Printf(color.Ize(color.Red, "Error: No sessions running\n"), n)
		} else if n > 1 {
			fmt.Printf(color.Ize(color.Green, "Stopped recording for %d sessions\n"), n)
		} else {
			fmt.Println(color.Ize(color.Green, "Stopped recording"))
		}
	} else {
		if n == 0 {
			fmt.Printf(color.Ize(color.Red, "Error: Name '%s' does not exist\n"), name)
		} else if n > 1 {
			fmt.Printf(color.Ize(color.Green, "Stopped recording for %d sessions named '%s'\n"), n, name)
		} else {
			fmt.Printf(color.Ize(color.Green, "Stopped recording for '%s'\n"), name)
		}
	}
	return nil
}

func extractSessions(rows *sql.Rows) []Session {
	var sessions []Session
	for rows.Next() {
		var session Session
		rows.Scan(&session.id, &session.name, &session.start, &session.finish)
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

	sessions := extractSessions(rows)
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

func totalDuration(sessions []Session) time.Duration {
	var totalDuration time.Duration
	for _, session := range sessions {
		duration := session.finish.Sub(session.start)
		totalDuration += duration
	}
	return totalDuration
}

func displayDuration(duration time.Duration, time string) {
	switch time {
	case "":
		fmt.Printf("Total duration: ")
	case "today":
		fmt.Printf("Total duration today: ")
	case "day":
		fmt.Printf("Total duration in last 24 hours: ")
	case "week":
		fmt.Printf("Total duration in last week: ")
	case "month":
		fmt.Printf("Total duration in last month: ")
	case "year":
		fmt.Printf("Total duration in last year: ")
	}
	fmt.Println(color.Ize(color.Green, durafmt.Parse(duration).LimitFirstN(2).String()))
}

func displayStats(db *sql.DB, period string) error {
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

	fmt.Printf("%d sessions\n", len(sessions))
	duration := totalDuration(sessions)
	displayDuration(duration, period)

	if period == "week" || period == "month" || period == "year" {
		var nDays int
		if period == "week" {
			nDays = 7
		} else if period == "month" {
			nDays = 30
		} else if period == "year" {
			nDays = 365
		}
		data := make([]float64, nDays)
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		for _, session := range sessions {
			day := time.Date(session.start.Year(), session.start.Month(), session.start.Day(), 0, 0, 0, 0, session.start.Location())
			daysAgo := int(today.Sub(day).Hours() / 24.0)
			sessionDuration := session.finish.Sub(session.start).Minutes()
			data[nDays-1-daysAgo] += sessionDuration
		}
		graph := asciigraph.Plot(data, asciigraph.Width(60))

		fmt.Printf("\n%s\n\n", graph)
	}

	return nil
}

func reset(db *sql.DB) error {
	stmt, err := db.Prepare("DROP TABLE IF EXISTS " + dbname)
	if err != nil {
		log.Printf("Error when preparing to drop table: %s\n", err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		log.Printf("Error when dropping table: %s\n", err)
		return nil
	}

	return nil
}

type Session struct {
	id     int
	name   string
	start  time.Time
	finish time.Time
}

func rowCount(rows *sql.Rows) int {
	count := 0
	for rows.Next() {
		count++
	}
	return count
}

func numCurrentSessions(db *sql.DB) (int, error) {
	rows, err := db.Query("SELECT * FROM clockin WHERE finish IS NULL")
	if err != nil {
		log.Printf("Current sessions failed with error: %s\n", err)
		return 0, err
	}

	count := rowCount(rows)
	return count, nil
}

func remindCurrentSessions(db *sql.DB) {
	n, err := numCurrentSessions(db)
	if err != nil {
		log.Printf("Getting number of current sessions failed with error: %s", err)
		return
	}
	if n > 1 {
		fmt.Println(color.Ize(color.Yellow, "Reminder: Session currently running"))
	}
}
func currentSessions(db *sql.DB) ([]Session, error) {
	rows, err := db.Query("SELECT * FROM clockin WHERE finish IS NULL")
	if err != nil {
		log.Printf("Current sessions failed with error: %s\n", err)
		return nil, err
	}

	sessions := extractSessions(rows)
	return sessions, nil
}

func printCurrentSession(session Session) {
	duration := color.Ize(color.Green, durafmt.Parse(time.Since(session.start)).LimitFirstN(2).String())

	if session.name == "" {
		fmt.Printf("[%d] running for %s\n", session.id, duration)
	} else {
		fmt.Printf("[%d - %s] running for %s\n", session.id, session.name, duration)
	}
}

func status(db *sql.DB) error {
	sessions, err := currentSessions(db)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions currently running.")
	} else {
		for _, session := range sessions {
			printCurrentSession(session)
		}
	}
	return nil
}

func getCommand() string {
	var command string
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	return command
}

func getAdditionalOption() string {
	var option string
	if len(os.Args) > 2 {
		option = os.Args[2]
	}
	return option
}

func checkValidTime(time string) bool {
	return time == "" || time == "today" || time == "day" || time == "week" || time == "month" || time == "year"
}

func getDBLoginDetails() (string, string) {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println(color.Ize(color.Red, "Error loading variables from .env file"))
	}
	username := os.Getenv("MYSQL_USERNAME")
	password := os.Getenv("MYSQL_PASSWORD")
	if username == "" || password == "" {
		fmt.Printf("Enter MySQL username: ")
		fmt.Scanln(&username)
		fmt.Printf("Enter MySQL password: ")
		fmt.Scanln(&password)

		// Save to .env file (overwrite any existing)
		f, err := os.Create("./.env")
		check(err)
		defer f.Close()
		_, err = fmt.Fprintf(f, "MYSQL_USERNAME=%s\nMYSQL_PASSWORD=%s", username, password)
		check(err)
	}

	return username, password
}

func main() {
	username, password := getDBLoginDetails()

	db, err := dbConnection(username, password)
	if err != nil {
		log.Printf("Error when getting database connection: %s\n", err)
		return
	}
	defer db.Close()

	err = createTable(db)
	if err != nil {
		log.Printf("Create table failed with error: %s\n", err)
		return
	}

	command := getCommand()

	switch command {
	case "start", "starting", "go":
		remindCurrentSessions(db)
		name := getAdditionalOption()
		err := startRecording(db, name)
		if err != nil {
			log.Printf("Start recording failed with error: %s\n", err)
			return
		}
	case "finish", "finished", "end", "stop", "halt":
		name := getAdditionalOption()
		err := finishRecording(db, name)
		if err != nil {
			log.Printf("Finish recording failed with error: %s\n", err)
			return
		}
	case "reset":
		err := reset(db)
		if err != nil {
			log.Printf("Data reset failed with error: %s\n", err)
			return
		}
	case "status", "info", "running":
		remindCurrentSessions(db)
		err := status(db)
		if err != nil {
			log.Printf("Data reset failed with error: %s\n", err)
			return
		}
	case "stats", "statistics":
		time := getAdditionalOption()
		if !checkValidTime(time) {
			fmt.Println(color.Ize(color.Red, "Error: Statistics time range invalid"))
			return
		}
		err := displayStats(db, time)
		if err != nil {
			log.Printf("Display stats failed with error: %s\n", err)
			return
		}
	}

	showTable(db)
}
