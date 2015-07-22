package zlog

import (
	"io"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/zvelo/zvelo-services/zenv"
)

// Flags is a convenience variable for cli arguments
var Flags = []cli.Flag{
	cli.StringFlag{
		Name:   "log-level, l",
		EnvVar: "LOG_LEVEL",
		Value:  "WARN",
		Usage:  "set log level (DEBUG, INFO, WARN, ERR)",
	},
	cli.StringFlag{
		Name:   "log-file, f",
		EnvVar: "LOG_FILE",
		Value:  "stderr",
		Usage:  "set log filename (stderr, stdout or any file name)",
	},
}

// ParseLevel parses a string and returns a level
func ParseLevel(level string) log.Level {
	l := log.WarnLevel

	switch strings.ToUpper(level) {
	case "ERR", "ERROR":
		l = log.ErrorLevel
	case "INFO":
		l = log.InfoLevel
	case "DEBUG":
		l = log.DebugLevel
	}

	return l
}

func parseFile(file string) (io.Writer, string, bool) {
	switch file {
	case "stderr", "STDERR":
		return os.Stderr, "stderr", false
	case "stdout", "STDOUT":
		return os.Stdout, "stdout", false
	default:
		f, err := os.OpenFile(file, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
		// file is only closed on os.Exit
		if err != nil {
			log.WithFields(log.Fields{
				"file": file,
				"err":  err,
			}).Fatal("could not open file")
		}
		return f, file, true
	}
}

// Init should be called once at the start of each application to enable
// the output log filtering. It depends on a cli context GlobalString
// "log-level" to set the MinLevel.
func Init(c *cli.Context, env zenv.ZEnv) {
	w, name, isFile := parseFile(c.String("log-file"))
	log.WithField("name", name).Info("log location")
	log.SetOutput(w)

	if env != zenv.Development {
		log.SetFormatter(&log.JSONFormatter{})
	} else if isFile {
		log.SetFormatter(&log.TextFormatter{
			DisableColors: true,
		})
	}

	ChangeLogLevel(c.String("log-level"))
}

// ChangeLogLevel updates the minimum log level
func ChangeLogLevel(minLevel string) {
	l := ParseLevel(minLevel)
	log.SetLevel(l)
	log.WithField("level", l).Info("log level changed")
}
