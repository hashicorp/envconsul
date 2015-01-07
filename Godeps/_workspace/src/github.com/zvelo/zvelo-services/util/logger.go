package util

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/codegangsta/cli"
	"github.com/codegangsta/negroni"
	"github.com/hashicorp/logutils"
)

var (
	loggerOnce sync.Once
	// LogFilter implements an io.Writer that writes to os.Stderr and filters
	// messages by logutils.LogLevel
	LogFilter io.Writer

	// NegroniLogger is a negroni logging middleware that implements
	// logutils.LogLevel
	NegroniLogger = negroni.HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		start := time.Now()
		next(rw, r)
		res := rw.(negroni.ResponseWriter)
		log.Printf("[INFO] %s %s %v %s in %v", r.Method, r.URL.Path, res.Status(), http.StatusText(res.Status()), time.Since(start))
	})

	// NegroniRecovery is a negroni recovery middleware that implements
	// logutils.LogLevel
	NegroniRecovery = &negroni.Recovery{
		Logger:     log.New(LogFilter, "[ERR] [negroni] ", 0),
		PrintStack: false,
		StackAll:   false,
		StackSize:  1024 * 8,
	}

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
		LogFilter = &logutils.LevelFilter{
			Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERR"},
			MinLevel: logutils.LogLevel(minLevel),
			Writer:   os.Stderr,
		}

		prefix := ""

		image := os.Getenv("IMAGE_NAME")
		if len(image) > 0 {
			prefix += fmt.Sprintf("image=%s ", image)
		}

		commit := os.Getenv("GIT_COMMIT")
		if len(commit) > 0 {
			prefix += fmt.Sprintf("commit=%s ", commit)
		}

		log.SetFlags(0)
		log.SetPrefix(prefix)
		log.SetOutput(LogFilter)
		log.Printf("[INFO] log level set to %s", minLevel)
	})
}
