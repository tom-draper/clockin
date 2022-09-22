# Clock In (In-Progress)

Clock In is a command-line tool that allows you to track your working periods. Start and stop the timer with two simple commands and view stats and trends and earn awards for good behaviour!.

## Getting Started

### MySQL

If MySQL is not installed on your machine, download it from the <a href="https://dev.mysql.com/downloads/mysql/">MySQL website</a>. During setup, ensure you note your username and password and copy them into the .env file.

### Build

Compile the program on your machine with:

```bash
go mod download
go build clockin.go
```

Run the executable with:

```bash
./clockin
```

## How to Use

### Starting a work session

To record a new work session, run:

```bash
clockin start
```

With multiple work sessions running at once, you can keep track of them with a name:

```bash
clockin start homework
```

### Finishing a work session

To finish a running work session, run:

```bash
clockin stop
```

With multiple work sessions running at once, you can specify your work session with its name identifier:

```bash
clockin stop homework
```

To stop all current running work sessions, run:

```bash
clockin stop all
```

### Check for running session

To check if a session is currently running, run:

```bash
clockin status
```

### Reset data

To delete all stored data, run:

```bash
clockin reset
```

### Statistics

A status summary can be displayed by running:

```bash
clockin stats
```

A time period can be specified.

```bash
clockin stats today
clockin stats day
clockin stats week
clockin stats month
clockin stats year
```

### Config

config.json contains configurable settings that affect the way clockin works.

#### timeout

An integer upper limit on the number of hours that can be considered a single working session. Once the given number of hours is reached, the work session will terminate. This can be helpful if you ever forget to finish a session. A value of null represents no upper limit, and a working session will only end once the finish command is run or the machine is shutdown. Defaults to null.

#### discardOnTimeout

A boolean value on whether a work session is discarded if the timeout limit is reached. Defaults to false.
