package main

import (
	"fmt"
	"log"
	"os"

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

func DisplayUsage() {
	fmt.Printf("clockin is a tool for recording work time.\n\nUsage:\n\n        clockin <command>\n\n        MySQL installation is required.\n\nThe commands are:\n\n        start          start timing a new work session\n        start <name>   start timing a new work session with an assigned name\n        finish         finish timing all currently running work sessions\n        finish <name>  finish timing a running work session, specified by its assigned name\n        running        list all currently running work sessions\n        stats          open statistics page\n        reset          delete all stored data\n")
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
		err := DisplayStatus(db)
		if err != nil {
			log.Printf("Data reset failed with error: %s\n", err)
			return
		}
	case "stats", "statistics":
		err := DisplayStats(db)
		if err != nil {
			log.Printf("Display stats failed with error: %s\n", err)
			return
		}
	case "", "help":
		DisplayUsage()
	case "show":
		ShowTable(db)
	default:
		fmt.Printf("clockin %s: unknown command\nRun 'clockin help' for usage", command)
	}

}
