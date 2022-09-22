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
	no, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error when fetching rows: %s", err)
		return nil, err
	}
	log.Printf("Rows affected: %d", no)

	db.Close()
	db, err = sql.Open("mysql", dsn(dbname))
	if err != nil {
		log.Printf("Error when opening database: %s", err)
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
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
	query := `CREATE TABLE IF NOT EXISTS clockin(id int primary key auto_increment, start datetime default CURRENT_TIMESTAMP, finish datetime default CURRENT_TIMESTAMP)`
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	res, err := db.ExecContext(ctx, query)
	if err != nil {
		log.Printf("Error when creating table: %s", err)
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error when getting rows affected: %s", err)
		return err
	}

	log.Printf("Rows affected: %d", rows)
	return nil
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
}
