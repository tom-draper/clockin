package main

import (
	"fmt"
	"log"
	"os"

	"github.com/TwiN/go-color"

	. "clockin/lib"

	_ "github.com/go-sql-driver/mysql"
)

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

func main() {
	db, err := OpenDatabase()
	if err != nil {
		return
	}

	command := getCommand()
	switch command {
	case "start", "starting", "go":
		name := getAdditionalOption()
		err := StartRecording(db, name)
		if err != nil {
			log.Printf("Start recording failed with error: %s\n", err)
			return
		}
		RemindCurrentSessions(db)
	case "finish", "finished", "end", "stop", "halt":
		name := getAdditionalOption()
		err := FinishRecording(db, name)
		if err != nil {
			log.Printf("Finish recording failed with error: %s\n", err)
			return
		}
		RemindCurrentSessions(db)
	case "reset":
		err := Reset(db)
		if err != nil {
			log.Printf("Data reset failed with error: %s\n", err)
			return
		}
	case "status", "info", "running":
		err := Status(db)
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
		err := DisplayStats(db, time)
		if err != nil {
			log.Printf("Display stats failed with error: %s\n", err)
			return
		}
	case "show":
		ShowTable(db)
	}

}
