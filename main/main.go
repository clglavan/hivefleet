package main

import (
	"os"

	hivefleet "github.com/clglavan/hivefleet"
)

func main() {
	confPath := os.Args[1]
	hivefleet.Run(confPath)
}
