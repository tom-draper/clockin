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
		Check(err)
		defer f.Close()
		_, err = fmt.Fprintf(f, "MYSQL_USERNAME=%s\nMYSQL_PASSWORD=%s", username, password)
		Check(err)
	}

	return username, password
}

func printCurrentSession(session Session) {
	duration := color.Ize(color.Green, durafmt.Parse(time.Since(session.Start)).LimitFirstN(2).String())

	if session.Name == "" {
		fmt.Printf("[%d] running for %s\n", session.ID, duration)
	} else {
		fmt.Printf("[%d - %s] running for %s\n", session.ID, session.Name, duration)
	}
}

func Status(db *sql.DB) error {
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

func FinishRecording(db *sql.DB, name string) error {
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

func numCurrentSessions(db *sql.DB) (int, error) {
	rows, err := db.Query("SELECT * FROM clockin WHERE finish IS NULL")
	if err != nil {
		log.Printf("Current sessions failed with error: %s\n", err)
		return 0, err
	}

	count := rowCount(rows)
	return count, nil
}

func RemindCurrentSessions(db *sql.DB) {
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

	sessions := ExtractSessions(rows)
	return sessions, nil
}

func OpenDatabase() (*sql.DB, error) {
	username, password := getDBLoginDetails()

	db, err := dbConnection(username, password)
	if err != nil {
		log.Printf("Error when getting database connection: %s\n", err)
		return nil, err
	}
	defer db.Close()

	err = createTable(db)
	if err != nil {
		log.Printf("Create table failed with error: %s\n", err)
		return nil, err
	}
	return db, nil
}
