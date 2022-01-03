package main

import (
	"fmt"
	"net/http"
	"os"
	"hive.glavan.tech/commander"
)

var port string

func init() {
	if len(os.Args) > 1 {
		port = ":" + os.Args[1]
	} else {
		port = ":3000"
	}
}

func main() {
	http.HandleFunc("/",commander.Commander)
	fmt.Println(http.ListenAndServe(port,nil))
}
