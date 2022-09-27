package clockin

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/TwiN/go-color"
	"github.com/hako/durafmt"
	"github.com/joho/godotenv"
)

const (
	hostname = "127.0.0.1:3306"
	dbname   = "clockin"
)

func getDBLoginDetails() (string, string, bool) {
	godotenv.Load(".env")
	fromEnv := true
	username := os.Getenv("MYSQL_USERNAME")
	password := os.Getenv("MYSQL_PASSWORD")
	if username == "" || password == "" {
		fmt.Println(color.Ize(color.Yellow, "MySQL login details required"))
		fmt.Printf("Username: ")
		fmt.Scanln(&username)
		fmt.Printf("Password: ")
		fmt.Scanln(&password)
		fromEnv = false
	}

	return username, password, fromEnv
}
func formatDuration(duration time.Duration) string {
	return durafmt.Parse(duration).LimitFirstN(2).String()
}

func printCurrentSession(session Session) {
	now := CurrentTime()
	duration := now.Sub(session.Start)
	durationStr := color.Ize(color.Green, formatDuration(duration))

	if session.Name == "" {
		fmt.Printf("[%d] running for %s\n", session.ID, durationStr)
	} else {
		fmt.Printf("[%d - %s] running for %s\n", session.ID, session.Name, durationStr)
	}
}

func Status(db *sql.DB) error {
	sessions, err := currentSessions(db)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println(color.Ize(color.Green, "No sessions currently running"))
	} else {
		if len(sessions) == 1 {
			fmt.Printf(color.Ize(color.Green, "%d session running\n"), len(sessions))
		} else {
			fmt.Printf(color.Ize(color.Green, "%d sessions running:\n"), len(sessions))
		}
		for _, session := range sessions {
			printCurrentSession(session)
		}
	}
	return nil
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
		fmt.Println(color.Ize(color.Red, "Error: Login details invalid"))
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

func ShowTable(db *sql.DB) error {
	res, err := db.Query("SELECT * FROM clockin")
	if err != nil {
		return err
	}

	for res.Next() {
		var session Session
		res.Scan(&session.ID, &session.Name, &session.Start, &session.Finish)
		name := session.Name
		if session.Name == "" {
			name = "none"
		}
		if session.Finish.IsZero() {
			fmt.Printf("%d %s %s %s\n", session.ID, name, session.Start, color.Ize(color.Yellow, session.Finish.String()))
		} else {
			fmt.Printf("%d %s %s %s\n", session.ID, name, session.Start, session.Finish)
		}
	}

	return nil
}

func StartRecording(db *sql.DB, name string) error {
	query := "INSERT INTO clockin(name, start, finish) VALUES (?, NOW(), NULL)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error when preparing SQL insert statement: %s\n", err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, name)
	if err != nil {
		log.Printf("Error when inserting row into products table: %s\n", err)
		return err
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	if name == "" {
		fmt.Printf(color.Ize(color.Green, "Started recording (%s)\n"), now)
	} else {
		fmt.Printf(color.Ize(color.Green, "Started recording %s (%s)\n"), name, now)
	}
	return nil
}

func sessionInList(session Session, sessions []Session) bool {
	for _, s := range sessions {
		if s.ID == session.ID {
			return true
		}
	}
	return false
}

func getUpdatedSessions(activeBefore []Session, activeAfter []Session) []Session {
	updatedSessions := []Session{}
	for _, session := range activeBefore {
		if !sessionInList(session, activeAfter) {
			updatedSessions = append(updatedSessions, session)
		}
	}
	return updatedSessions
}

func getUpdatedSession(activeBefore []Session, activeAfter []Session) Session {
	for _, session := range activeBefore {
		if !sessionInList(session, activeAfter) {
			return session
		}
	}
	return Session{}
}

func FinishRecording(db *sql.DB, name string) error {
	activeSessionsBefore, err := currentSessions(db)

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

	now := CurrentTime()
	if name == "" {
		if n == 0 {
			fmt.Println(color.Ize(color.Red, "No sessions running"))
		} else if n > 1 {
			fmt.Printf(color.Ize(color.Green, "Stopped recording for %d sessions\n"), n)
		} else {
			activeSessionsAfter, err := currentSessions(db)
			Check(err)
			updated := getUpdatedSession(activeSessionsBefore, activeSessionsAfter)
			updated.Finish = now
			duration := calcDuration(updated)
			fmt.Printf(color.Ize(color.Green, "Stopped recording (%s)\n"), formatDuration(duration))
		}
	} else {
		if n == 0 {
			fmt.Printf(color.Ize(color.Red, "Name '%s' does not exist\n"), name)
		} else if n > 1 {
			fmt.Printf(color.Ize(color.Green, "Stopped recording for %d sessions named '%s'\n"),
				n, name)
		} else {
			activeSessionsAfter, err := currentSessions(db)
			Check(err)
			updated := getUpdatedSession(activeSessionsBefore, activeSessionsAfter)
			updated.Finish = now
			duration := calcDuration(updated)
			fmt.Printf(color.Ize(color.Green, "Stopped recording for '%s' (%s)\n"),
				name, formatDuration(duration))
		}
	}
	return nil
}

func Reset(db *sql.DB) error {
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

func rowCount(rows *sql.Rows) int {
	count := 0
	for rows.Next() {
		count++
	}
	return count
}

func getSession(db *sql.DB, sessionID int) Session {
	var session Session
	err := db.QueryRow("SELECT * FROM clockin WHERE is=?").Scan(&session)
	if err != nil {
		return Session{}
	}

	fmt.Println(session)

	return session
}

func NumActiveSessions(db *sql.DB) (int, error) {
	rows, err := db.Query("SELECT * FROM clockin WHERE finish IS NULL")
	if err != nil {
		log.Printf("Finding number of active sessions failed with error: %s\n", err)
		return 0, err
	}

	count := rowCount(rows)
	return count, nil
}

func RemindCurrentSessions(db *sql.DB) {
	n, err := NumActiveSessions(db)
	if err != nil {
		log.Printf("Getting number of current sessions failed with error: %s\n", err)
		return
	}
	if n > 1 {
		fmt.Printf(color.Ize(color.Yellow, "Reminder: %d sessions currently running\n"), n)
	}
}

func currentSessions(db *sql.DB) ([]Session, error) {
	rows, err := db.Query("SELECT * FROM clockin WHERE finish IS NULL")
	if err != nil {
		log.Printf("Current sessions failed with error: %s\n", err)
		return nil, err
	}

	sessions := ExtractSessions(rows)
	return sessions, nil
}

func OpenDatabase() (*sql.DB, error) {
	username, password, fromEnv := getDBLoginDetails()

	db, err := dbConnection(username, password)
	if err != nil {
		return nil, err
	}

	if !fromEnv {
		fmt.Println(color.Ize(color.Green, "Login successful\n"))
		// Save details to .env file (overwrite any existing)
		f, err := os.Create("./.env")
		Check(err)
		defer f.Close()
		_, err = fmt.Fprintf(f, "MYSQL_USERNAME=%s\nMYSQL_PASSWORD=%s", username, password)
		Check(err)
	}

	err = createTable(db)
	if err != nil {
		log.Printf("Create table failed with error: %s\n", err)
		return nil, err
	}
	return db, nil
}
