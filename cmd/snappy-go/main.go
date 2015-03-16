package main

import (
	"fmt"
	"os"
	"errors"
	"syscall"
	"log"
	"log/syslog"

	"github.com/jessevdk/go-flags"
)

// fixed errors the command can return
var ErrRequiresRoot = errors.New("command requires sudo (root)")

// Name used in the prefix for all logged messages
const logIdentifier = "snappy"

type options struct {
	// No global options yet
}

var optionsData options

var parser = flags.NewParser(&optionsData, flags.Default)

func setupLogger () {
	// forward all log messages to syslog
	writer, err := syslog.New(syslog.LOG_INFO|syslog.LOG_USER, logIdentifier)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to create syslog connection");
	}

	log.SetOutput(writer)
	log.SetPrefix("ERROR:")
}

func init() {
	setupLogger()
}

func LogError(err error) error {
	log.Print(err)
	return err
}

func main() {
	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}

func isRoot() bool {
	return syscall.Getuid() == 0
}
