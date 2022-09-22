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
	"github.com/hako/durafmt"
)

const (
	username = "root"
	password = "root"
	hostname = "127.0.0.1:3306"
	dbname   = "clockin"
)

func dsn(dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", username, password, hostname, dbName)
}

func rowsAffected(res sql.Result) (int64, error) {
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error when getting rows affected: %s", err)
		return 0, err
	}
	log.Printf("Rows affected: %d", rows)
	return rows, nil
}

func dbConnection() (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn(""))
	if err != nil {
		log.Printf("Error when opening database: %s", err)
		return nil, err
	}

	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	res, err := db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+dbname)
	if err != nil {
		log.Printf("Error when creating database: %s", err)
		return nil, err
	}

	rowsAffected(res)

	db.Close()
	db, err = sql.Open("mysql", dsn(dbname))
	if err != nil {
		log.Printf("Error when opening database: %s", err)
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Minute * 5)

	log.Printf("Connection established")
	return db, nil
}

func createTable(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS clockin(id int primary key auto_increment, name varchar(100), start datetime default CURRENT_TIMESTAMP, finish datetime)`
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	_, err := db.ExecContext(ctx, query)
	if err != nil {
		log.Printf("Error when creating table: %s", err)
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
		log.Printf("%d %s %s %s", session.id, session.name, session.start, session.finish)
	}

	return nil
}

func startRecording(db *sql.DB, name string) error {
	query := "INSERT INTO clockin(name, start, finish) VALUES (?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error when preparing SQL insert statement: %s", err)
		return err
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, name, time.Now().UTC(), nil)
	if err != nil {
		log.Printf("Error when inserting row into products table: %s", err)
		return err
	}

	rowsAffected(res)

	return nil
}

func finishRecording(db *sql.DB, name string) error {
	var query string
	if name == "all" {
		query = "UPDATE clockin set finish=? WHERE finish is NULL"
	} else {
		query = "UPDATE clockin set finish=? WHERE finish is NULL AND name=?"
	}
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error when preparing SQL update statement: %s", err)
		return err
	}
	defer stmt.Close()

	var res sql.Result
	if name == "all" {
		res, err = stmt.ExecContext(ctx, time.Now().UTC())
	} else {
		res, err = stmt.ExecContext(ctx, time.Now().UTC(), name)
	}
	if err != nil {
		log.Printf("Error when inserting row into products table: %s", err)
		return err
	}

	rowsAffected(res)

	return nil
}

func extractSessions(rows *sql.Rows) []Session {
	var sessions []Session
	for rows.Next() {
		var session Session
		rows.Scan(&session.id, &session.name, &session.start, &session.finish)
		log.Printf("Session %d %s %s %s", session.id, session.name, session.start, session.finish)
		sessions = append(sessions, session)
	}
	return sessions
}

func getSessionsToday(db *sql.DB) ([]Session, error) {
	rows, err := db.Query("SELECT * FROM clockin WHERE FINISH IS NOT NULL AND start BETWEEN NOW() AND CURRENT_DATE() + INTERVAL 1 DAY)")
	if err != nil {
		return nil, err
	}

	sessions := extractSessions(rows)
	return sessions, nil
}

func getSessionsDay(db *sql.DB) ([]Session, error) {
	rows, err := db.Query("SELECT * FROM clockin WHERE finish IS NOT NULL AND start >= DATE_SUB(NOW(), INTERVAL 1 DAY)")
	if err != nil {
		return nil, err
	}

	sessions := extractSessions(rows)
	return sessions, nil
}

func getSessionsWeek(db *sql.DB) ([]Session, error) {
	rows, err := db.Query("SELECT * FROM clockin WHERE finish IS NOT NULL AND start >= DATE_SUB(NOW(), INTERVAL 1 WEEK)")
	if err != nil {
		return nil, err
	}

	sessions := extractSessions(rows)
	return sessions, nil
}

func getSessionsMonth(db *sql.DB) ([]Session, error) {
	rows, err := db.Query("SELECT * FROM clockin WHERE finish IS NOT NULL AND start >= DATE_SUB(NOW(), INTERVAL 1 MONTH)")
	if err != nil {
		return nil, err
	}

	sessions := extractSessions(rows)
	return sessions, nil
}

func getSessionsYear(db *sql.DB) ([]Session, error) {
	rows, err := db.Query("SELECT * FROM clockin WHERE finish IS NOT NULL AND start >= DATE_SUB(NOW(), INTERVAL 1 YEAR)")
	if err != nil {
		return nil, err
	}

	sessions := extractSessions(rows)
	return sessions, nil
}

func totalDuration(sessions []Session) time.Duration {
	var totalDuration time.Duration
	for _, session := range sessions {
		duration := session.finish.Sub(session.start)
		totalDuration += duration
	}
	return totalDuration
}

func displaySessionsStats(sessions []Session) error {
	duration := totalDuration(sessions)
	log.Println("Duration:", duration)
	return nil
}

func displayStats(db *sql.DB, time string) error {
	var sessions []Session
	var err error
	switch time {
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
		log.Printf("Sessions in time range failed with error: %s", err)
		return err
	}
	displaySessionsStats(sessions)
	return nil
}

func reset(db *sql.DB) error {
	stmt, err := db.Prepare("DROP TABLE IF EXISTS " + dbname)
	if err != nil {
		log.Printf("Error when preparing to drop table: %s", err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		log.Printf("Error when dropping table: %s", err)
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

func currentSessions(db *sql.DB) ([]Session, error) {
	rows, err := db.Query("SELECT * FROM clockin WHERE finish IS NULL")
	if err != nil {
		log.Printf("Current sessions failed with error: %s", err)
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

func main() {
	db, err := dbConnection()
	if err != nil {
		log.Printf("Error when getting database connection: %s", err)
		return
	}
	defer db.Close()

	err = createTable(db)
	if err != nil {
		log.Printf("Create table failed with error: %s", err)
		return
	}

	command := getCommand()

	switch command {
	case "start", "starting":
		name := getAdditionalOption()
		err := startRecording(db, name)
		if err != nil {
			log.Printf("Start recording failed with error: %s", err)
			return
		}
	case "finish", "finished", "end":
		name := getAdditionalOption()
		err := finishRecording(db, name)
		if err != nil {
			log.Printf("Finish recording failed with error: %s", err)
			return
		}
	case "reset":
		err := reset(db)
		if err != nil {
			log.Printf("Data reset failed with error: %s", err)
			return
		}
	case "status", "info", "running":
		err := status(db)
		if err != nil {
			log.Printf("Data reset failed with error: %s", err)
			return
		}
	case "stats", "statistics":
		time := getAdditionalOption()
		err := displayStats(db, time)
		if err != nil {
			log.Printf("Display stats failed with error: %s", err)
			return
		}
	}

	showTable(db)
}
