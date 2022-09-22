package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	username = "root"
	password = "root"
	hostname = "127.0.0.1:3306"
	dbname   = "clockin"
)

func dsn(dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s", username, password, hostname, dbName)
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

	ctx, cancelfunc = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	err = db.PingContext(ctx)
	if err != nil {
		log.Printf("Errors pinging database: %s", err)
		return nil, err
	}
	log.Printf("Connection established")
	return db, nil
}

func createTable(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS clockin(id int primary key auto_increment, name varchar(100) default NULL, start datetime default CURRENT_TIMESTAMP, finish datetime)`
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
		var id int64
		var name string
		var start time.Time
		var finish time.Time
		res.Scan(&id, &name, &start, &finish)
		log.Printf("%d %s %s %s", id, name, start, finish)
	}

	return nil
}

func currentWorkingID(db *sql.DB) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM clockin WHERE finish IS NULL").Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func startRecording(db *sql.DB) error {
	query := "INSERT INTO clockin(name, start, finish) VALUES (?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error when preparing SQL statement: %s", err)
		return err
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, "test", time.Now(), nil)
	if err != nil {
		log.Printf("Error when inserting row into products table: %s", err)
		return err
	}

	rowsAffected(res)

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

func main() {
	db, err := dbConnection()
	if err != nil {
		log.Printf("Error when getting database connection: %s", err)
		return
	}
	defer db.Close()

	reset(db)
	err = createTable(db)
	if err != nil {
		log.Printf("Create table failed with error: %s", err)
		return
	}

	err = startRecording(db)
	if err != nil {
		log.Printf("Start recording failed with error: %s", err)
		return
	}

	showTable(db)

	id, err := currentWorkingID(db)
	if err != nil {
		log.Printf("Getting current working ID failed with error: %s", err)
		return
	}
	log.Printf("Current working ID: %d", id)
}
