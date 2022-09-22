# Clock In (In-Progress)

## Getting Started

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

### Finishing a work session

To finish a running work session, run:

```bash
clockin finishing
```

### Check for running sesssion

To check if a session is currently running, run:

```bash
clockin status
```

### Statistics

A status summary can be displayed by running:

```bash
clockin stats
```

To view statistics for the current day, run:

```bash
clockin stats --today
```

To view statistics for the last month, run:

```bash
clockin stats --month
```

To view statistics for the last year, run:

```bash
clockin stats --year
```

### Config

config.json contains configurable settings that affect the way clockin works.

### timeout

An integer upper limit on the number of hours that can be considered a single working session. Once the given number of hours is reached, the work session will terminate. This can be helpful if you ever forget to finish a session. A value of null represents no upper limit, and a . Defaults to null.

```json
{
    "timeout": 30
}
```

### discardOnTimeout

A boolean value on whether a work session is discarded if the timeout limit is reached. Defaults to false.
