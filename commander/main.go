package commander

import (
	"bytes"
	// "context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"time"
	// "encoding/json"
	// "cloud.google.com/go/storage"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789")

func randStr(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func Commander(w http.ResponseWriter, r *http.Request) {
	rand.Seed(time.Now().UnixNano())

	local := r.URL.Query().Get("local")

	// if(local != "1"){
	// 	fmt.Println("Export the credentials to path")
	// 	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./serverless_function_source_code/credentials.json")
	// }
	url := r.URL.Query().Get("url")
	concurrency := r.URL.Query().Get("concurrency")
	number := r.URL.Query().Get("number")

	bombardierPath := "./serverless_function_source_code/bombardier"
	if local == "1" {
		bombardierPath = "./../bombardier-mac"
	}
	// fmt.Println("Error: ", url,concurrency,number)
	bombardier := exec.Command(bombardierPath, "-c", concurrency, "-n", number, url, "-p", "r", "-o", "json")
	buf := bytes.Buffer{}
	bombardier.Stdout = &buf
	// auth.Stdout = os.Stdout
	bombardier.Stderr = os.Stderr
	if err := bombardier.Run(); err != nil {
		fmt.Println("Error: ", err)
	}

	s := buf.String()

	//Marshal or convert user object back to json and write to response
	// response, err := json.Marshal(s)
	// if err != nil{
	// 	panic(err)
	// }
	// fmt.Println("test ",s)
	// json.NewEncoder(w).Encode(s)

	//Set Content-Type header so that clients will know how to read response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	//Write json response back to response
	w.Write([]byte(s))

	// return s;

	// ctx := context.Background()
	// client, err := storage.NewClient(ctx)
	// if err != nil {
	// 	fmt.Println("Error: ", err)
	// }
	// wc := client.Bucket("hive-fleet-results").Object("results/results-" + randStr(5) + ".json").NewWriter(ctx)
	// wc.ContentType = "text/plain"
	// wc.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
	// if _, err := wc.Write([]byte(s)); err != nil {
	// 	// TODO: handle error.
	// 	// Note that Write may return nil in some error situations,
	// 	// so always check the error from Close.
	// 	fmt.Println("Error: ", err)
	// }
	// if err := wc.Close(); err != nil {
	// 	fmt.Println("Error: ", err)
	// }
	// fmt.Println("updated object:", wc.Attrs())

}
