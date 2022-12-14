<p align="center">
	<img width="550px" src="https://user-images.githubusercontent.com/41476809/192119791-831cec36-dab0-4cb0-afc1-1ba12389475f.png">
</p>

A command-line tool for tracking your working periods. Start and stop the timer with two simple commands, and then open the built-in statistical summary to identify patterns and trends!

## Getting Started

### Install Dependencies

A local <a href="https://dev.mysql.com/downloads/mysql/">MySQL</a> server is required by clockin to store session data on your machine. During setup, ensure you make note of your MySQL username and password and copy them into the .env file or enter them straight into the command line during the first run of the program.

Download Go dependencies with:

```bash
go mod download
```

### Build

Compile the program on your machine with:

```bash
go build clockin.go
```

To make the executable runnable from anywhere, add the directory as a PATH environment variable.

## How to Use

### Starting a work session

To start recording a new work session, run:

```bash
clockin start
```

You can keep track of multiple work sessions running at once by providing a name identifier:

```bash
clockin start homework
```

### Finishing a work session

To finish recording for all currently running work sessions, run:

```bash
clockin stop
```

You can specify a particular work session by its name identifier:

```bash
clockin stop homework
```

### Show running sessions

To list all currently running work sessions, run:

```bash
clockin running
```

### Reset data

To delete all stored data, run:

```bash
clockin reset
```

### Statistics

A statistical summary of how you've spent your time working can be displayed by running:

```bash
clockin stats
```
<!---
### Config

config.json contains configurable settings that affect the way clockin works.

#### timeout

An integer upper limit on the number of hours that can be considered a single working session. Once the given number of hours is reached, the work session will terminate. This can be helpful if you ever forget to finish a session. A value of null represents no upper limit, and a working session will only end once the finish command is run or the machine is shutdown. Defaults to null.

#### discardOnTimeout

A boolean value on whether a work session is discarded if the timeout limit is reached. Defaults to false.
-->
