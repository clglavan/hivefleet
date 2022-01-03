package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

type Result struct {
	BytesRead        int     `json:"bytesRead"`
	BytesWritten     int     `json:"bytesWritten"`
	TimeTakenSeconds float64 `json:"timeTakenSeconds"`
	Req1Xx           int     `json:"req1xx"`
	Req2Xx           int     `json:"req2xx"`
	Req3Xx           int     `json:"req3xx"`
	Req4Xx           int     `json:"req4xx"`
	Req5Xx           int     `json:"req5xx"`
	Others           int     `json:"others"`
	Latency          struct {
		Mean   float64 `json:"mean"`
		Stddev float64 `json:"stddev"`
		Max    float64 `json:"max"`
	} `json:"latency"`
	Rps struct {
		Mean        float64 `json:"mean"`
		Stddev      float64 `json:"stddev"`
		Max         float64 `json:"max"`
		Percentiles struct {
			Num50 float64 `json:"50"`
			Num75 float64 `json:"75"`
			Num90 float64 `json:"90"`
			Num95 float64 `json:"95"`
			Num99 float64 `json:"99"`
		} `json:"percentiles"`
	} `json:"rps"`
}

type Report struct {
	Spec struct {
		NumberOfConnections int    `json:"numberOfConnections"`
		TestType            string `json:"testType"`
		NumberOfRequests    int    `json:"numberOfRequests"`
		Method              string `json:"method"`
		URL                 string `json:"url"`
		Body                string `json:"body"`
		Stream              bool   `json:"stream"`
		TimeoutSeconds      int    `json:"timeoutSeconds"`
		Client              string `json:"client"`
	} `json:"spec"`
	Result Result `json:"result"`
}

type conf struct {
	Clients          int
	Credentials      string
	Deploy_function  int
	Function_memory  string
	Function_timeout string
	Function_region  string
	Concurrency      string
	Number           string
	Url              string
	Local            int
}

func (c *conf) getConf() *conf {

	yamlFile, err := ioutil.ReadFile("config.yml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

// func init() {

// }

func main() {
	var c conf
	c.getConf()

	// fmt.Println(c.Deploy_function)

	if c.Local == 0 {
		fmt.Println("Export the credentials to path")
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", c.Credentials)

		// Authenticating the service account
		fmt.Println("Activating credentials from " + c.Credentials)
		auth := exec.Command("gcloud", "auth", "activate-service-account", "--key-file="+c.Credentials)
		auth.Stdout = os.Stdout
		auth.Stderr = os.Stderr
		if err := auth.Run(); err != nil {
			fmt.Println("Error: ", err)
		}

		// create the results bucket if it doesn't exist
		fmt.Println("Creating the results bucket")
		bucket := exec.Command("gsutil", "mb", "gs://hive-fleet-results")
		bucket.Stdout = os.Stdout
		bucket.Stderr = os.Stderr
		if err := bucket.Run(); err != nil {
			fmt.Println("Error: ", err)
		}

		fmt.Println("Cleanup results bucket")
		cleanup := exec.Command("gsutil", "rm", "gs://hive-fleet-results/results/*")
		// cleanup.Stdout = os.Stdout
		// cleanup.Stderr = os.Stderr
		if err := cleanup.Run(); err != nil {
			fmt.Println("Error: ", err)
		}
	}
	// gsutil mb gs://BUCKET_NAME
	// gsutil du -s gs://BUCKET_NAME

	// gsutil cp OBJECT_LOCATION gs://DESTINATION_BUCKET_NAME/

	// Getting the current working directory
	currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(c.Deploy_function)
	if c.Deploy_function == 1 && c.Local == 0 {
		fmt.Println("Deploying the function is set to true")
		// Deploy
		//"--allow-unauthenticated"
		deploy := exec.Command("gcloud", "functions", "deploy", "Commander", "--runtime", "go116", "--trigger-http", "--memory="+c.Function_memory, "--timeout="+c.Function_timeout)
		deploy.Dir = currentDir + "/commander"
		deploy.Stdout = os.Stdout
		deploy.Stderr = os.Stderr
		fmt.Println("Deploying function from " + deploy.Dir)
		if err := deploy.Run(); err != nil {
			fmt.Println("Error: ", err)
		}
		// sleep a bit, wait for the function to update
		// time.Sleep(20 * time.Second)
	}

	// Get the authorization token
	token := exec.Command("gcloud", "auth", "print-identity-token")
	buf := bytes.Buffer{}
	token.Stdout = &buf
	token.Stderr = os.Stderr
	if err := token.Run(); err != nil {
		fmt.Println("Error: ", err)
	}
	tokenString := strings.TrimSpace(buf.String())
	fmt.Printf("The Token: %s\n", tokenString)

	// Get the project id
	projectId := exec.Command("gcloud", "config", "get-value", "project")
	bufProjectId := bytes.Buffer{}
	projectId.Stdout = &bufProjectId
	projectId.Stderr = os.Stderr
	if err := projectId.Run(); err != nil {
		fmt.Println("Error: ", err)
	}
	projectIdString := strings.TrimSpace(bufProjectId.String())
	fmt.Printf("Project id: %s\n", projectIdString)

	fmt.Println("Triggering " + fmt.Sprint(c.Clients) + " functions")
	urlPath := "https://" + c.Function_region + "-" + projectIdString + ".cloudfunctions.net/Commander?concurrency=" + c.Concurrency + "&number=" + c.Number + "&url=" + url.QueryEscape(c.Url)

	if c.Local == 1 {
		urlPath = "http://localhost:3000/?concurrency=" + c.Concurrency + "&number=" + c.Number + "&url=" + url.QueryEscape(c.Url) + "&local=1"
	}

	client := http.Client{}
	req, err := http.NewRequest("GET", urlPath, nil)
	if err != nil {
		fmt.Println("Error: ", err)
	}

	req.Header = http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer " + tokenString},
	}

	var wg sync.WaitGroup

	var finalReport Result

	reports := make([]Report, c.Clients)

	for i := 0; i < c.Clients; i++ {
		fmt.Println("iteration: ", i)

		wg.Add(1)
		go func(i int) {
			res, err := client.Do(req)
			if err != nil {
				fmt.Println("Error: ", err)
			}
			fmt.Print(res.StatusCode, " ")
			if res.StatusCode == http.StatusOK {
				bodyBytes, err := ioutil.ReadAll(res.Body)
				if err != nil {
					log.Fatal(err)
				}
				bodyString := string(bodyBytes)
				fmt.Println(bodyString)
				// res := Report{}
				// json.Unmarshal([]byte(bodyString), &reports[i])
				// json.Unmarshal([]byte(bodyString), &res)
				json.Unmarshal([]byte(bodyString), &reports[i])
				// fmt.Println()

				finalReport.BytesRead = (finalReport.BytesRead + reports[i].Result.BytesRead) / 2
				finalReport.BytesWritten = (finalReport.BytesWritten + reports[i].Result.BytesWritten) / 2
				finalReport.TimeTakenSeconds = math.Round(((finalReport.TimeTakenSeconds+reports[i].Result.TimeTakenSeconds)/2)*100) / 100
				finalReport.Req1Xx = (finalReport.Req1Xx + reports[i].Result.Req1Xx) / 2
				finalReport.Req2Xx = (finalReport.Req2Xx + reports[i].Result.Req2Xx) / 2
				finalReport.Req3Xx = (finalReport.Req3Xx + reports[i].Result.Req3Xx) / 2
				finalReport.Req4Xx = (finalReport.Req4Xx + reports[i].Result.Req4Xx) / 2
				finalReport.Req5Xx = (finalReport.Req5Xx + reports[i].Result.Req5Xx) / 2
				finalReport.Others = (finalReport.Others + reports[i].Result.Others) / 2
				finalReport.Latency.Mean = math.Round(((finalReport.Latency.Mean+reports[i].Result.Latency.Mean)/2)*100) / 100
				finalReport.Latency.Stddev = math.Round(((finalReport.Latency.Stddev+reports[i].Result.Latency.Stddev)/2)*100) / 100
				finalReport.Latency.Max = math.Round(((finalReport.Latency.Max+reports[i].Result.Latency.Max)/2)*100) / 100
				finalReport.Rps.Mean = math.Round(((finalReport.Rps.Mean+reports[i].Result.Rps.Mean)/2)*100) / 100
				finalReport.Rps.Stddev = math.Round(((finalReport.Rps.Stddev+reports[i].Result.Rps.Stddev)/2)*100) / 100
				finalReport.Rps.Max = math.Round(((finalReport.Rps.Max+reports[i].Result.Rps.Max)/2)*100) / 100
				finalReport.Rps.Percentiles.Num50 = math.Round(((finalReport.Rps.Percentiles.Num50+reports[i].Result.Rps.Percentiles.Num50)/2)*100) / 100
				finalReport.Rps.Percentiles.Num75 = math.Round(((finalReport.Rps.Percentiles.Num75+reports[i].Result.Rps.Percentiles.Num75)/2)*100) / 100
				finalReport.Rps.Percentiles.Num90 = math.Round(((finalReport.Rps.Percentiles.Num90+reports[i].Result.Rps.Percentiles.Num90)/2)*100) / 100
				finalReport.Rps.Percentiles.Num95 = math.Round(((finalReport.Rps.Percentiles.Num95+reports[i].Result.Rps.Percentiles.Num95)/2)*100) / 100
				finalReport.Rps.Percentiles.Num99 = math.Round(((finalReport.Rps.Percentiles.Num99+reports[i].Result.Rps.Percentiles.Num99)/2)*100) / 100
			}
			wg.Done()
		}(i)
	}

	// fmt.Println(reports)
	wg.Wait()
	for i := 0; i < c.Clients; i++ {
		fmt.Println(reports[i])
	}
	fmt.Println("----------------- final report -------------------")
	fmt.Println(finalReport)

	t, err := template.ParseFiles("report_template.html")
	if err != nil {
		log.Print(err)
		return
	}

	f, err := os.Create("report.html")
	if err != nil {
		log.Println("create file: ", err)
		return
	}

	err = t.Execute(f, finalReport)
	if err != nil {
		log.Print("execute: ", err)
		return
	}
	f.Close()

	// curl -X POST "https://YOUR_REGION-YOUR_PROJECT_ID.cloudfunctions.net/commander" -H "Content-Type:application/json"'

	// report
	// ctx := context.Background()
	// client, err := storage.NewClient(ctx)
	// if err != nil {
	// 	fmt.Println("Error: ", err)
	// }
	// // + fmt.Sprint(time.Now().Unix()) + make report unique
	// wc := client.Bucket("hive-fleet-results").Object("results.json").NewWriter(ctx)
	// wc.ContentType = "text/plain"
	// wc.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
	// if _, err := wc.Write([]byte("[")); err != nil {
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
