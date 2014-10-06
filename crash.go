// +build ignore

package main

import (
	"os"
	"strconv"
	"time"
)

func main() {
	ts, _ := strconv.ParseInt(os.Args[1], 10, 64)

	if !time.Unix(ts, 0).Before(time.Now()) {
		os.Exit(1)
	}
}
