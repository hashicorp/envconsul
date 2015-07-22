package zenv

import (
	"strings"

	"github.com/codegangsta/cli"
)

// ZEnv is the runtime environment
type ZEnv int

const (
	// Development environment
	Development ZEnv = iota
	// Test environment
	Test
	// Integration environment
	Integration
	// Production environment
	Production
)

// Flag is the cli option
var Flag = cli.StringFlag{
	Name:   "zvelo-env, e",
	EnvVar: "ZVELO_ENV",
	Usage:  "runtime environment ['development', 'test', 'integration', 'production']",
	Value:  "development",
}

// Init parses env and returns the corresponding ZEnv
func Init(c *cli.Context) ZEnv {
	switch strings.ToUpper(c.String("zvelo-env")) {
	case "TEST":
		return Test
	case "INTEGRATION":
		return Integration
	case "PRODUCTION":
		return Production
	default:
		return Development
	}
}

//go:generate stringer -type=ZEnv
