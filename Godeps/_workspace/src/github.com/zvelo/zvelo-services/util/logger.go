package util

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/codegangsta/cli"
	"github.com/hashicorp/logutils"
)

var (
	loggerOnce sync.Once

	// logFilter implements an io.Writer that writes to os.Stderr and filters
	// messages by logutils.LogLevel
	logFilter io.Writer

	// LogLevelFlag is a convenience variable for a simple log-level cli argument
	LogLevelFlag = cli.StringFlag{
		Name:   "log-level, l",
		EnvVar: "LOG_LEVEL",
		Value:  "WARN",
		Usage:  "set log level (DEBUG, INFO, WARN, ERR)",
	}
)

// InitLogger should be called once at the start of each application to enable
// the output log filtering. It depends on a cli context GlobalString
// "log-level" to set the MinLevel.
func InitLogger(minLevel string) {
	loggerOnce.Do(func() {
		logFilter = &logutils.LevelFilter{
			Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERR"},
			MinLevel: logutils.LogLevel(minLevel),
			Writer:   os.Stderr,
		}

		// $IMAGE_NAME and $GIT_COMMIT are no longer used as a prefix because
		// they are set as a tag/field in the fluent metadata

		log.SetFlags(0)
		log.SetOutput(logFilter)
		log.Printf("[INFO] log level set to %s", minLevel)
	})
}

// ChangeLogLevel updates the minimum log level
func ChangeLogLevel(minLevel string) {
	logFilter.(*logutils.LevelFilter).MinLevel = logutils.LogLevel(minLevel)
}

// InitLoggerCli should be called once at the start of each application to enable
// the output log filtering. It depends on a cli context GlobalString
// "log-level" to set the MinLevel.
func InitLoggerCli(c *cli.Context) {
	InitLogger(c.GlobalString("log-level"))
}

// NewLogger returns a pointer to an object that implements the log.Logger
// interface but otherwise works the same as the standard logger.
func NewLogger(level string) *log.Logger {
	if logFilter == nil {
		log.Fatalln("Must call InitLogger first!")
	}
	return log.New(logFilter, fmt.Sprintf("[%s] ", level), 0)
}
