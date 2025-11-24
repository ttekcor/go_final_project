package tests

import (
	"os"
	"strconv"
)

var Port = func() int {
	if v := os.Getenv("TODO_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			return p
		}
	}
	return 7541
}()

var DBFile = "../scheduler.db"
var FullNextDate = true
var Search = true
var Token = ``
