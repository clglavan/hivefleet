package hivefleet

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
	"path"
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
	Debug            int
}

func (c *conf) getConf() *conf {

	if len(os.Args) < 2 {
		fmt.Println("For the use of this program it is mandatory to have give the path to a config.yml file")
		fmt.Println("Please consult https://github.com/clglavan/hive-fleet for an example")
		os.Exit(1)
	}
	arg := os.Args[1]
	fmt.Println(arg)

	yamlFile, err := os.ReadFile(arg)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		fmt.Println("Unmarshal: ", err)
		os.Exit(1)
	}

	return c
}

func exit() {

	// cleanup before exit with error

	e, err := os.Executable()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	clean := os.RemoveAll(path.Dir(e) + "/code")
	if clean != nil {
		fmt.Println(clean)
		os.Exit(1)
	}

	os.Exit(1)
}

func Run() {
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
			os.Exit(1)
		}

		// TODO - upload results to bucket
		// // create the results bucket if it doesn't exist
		// fmt.Println("Creating the results bucket")
		// bucket := exec.Command("gsutil", "mb", "gs://hive-fleet-results")
		// bucket.Stdout = os.Stdout
		// bucket.Stderr = os.Stderr
		// if err := bucket.Run(); err != nil {
		// 	fmt.Println("Error: ", err)
		// }

		// fmt.Println("Cleanup results bucket")
		// cleanup := exec.Command("gsutil", "rm", "gs://hive-fleet-results/results/*")
		// // cleanup.Stdout = os.Stdout
		// // cleanup.Stderr = os.Stderr
		// if err := cleanup.Run(); err != nil {
		// 	fmt.Println("Error: ", err)
		// }
	}

	// Getting the current working directory
	// currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	// if err != nil {
	// 	log.Fatal(err)
	// }

	if c.Deploy_function == 1 && c.Local == 0 {

		fmt.Println("Deploying the function is set to true")

		e, err := os.Executable()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// os.Exit(3)

		clone := exec.Command("git", "clone", "https://github.com/clglavan/hive-fleet.git", "code")
		clone.Dir = path.Dir(e)
		clone.Stdout = os.Stdout
		clone.Stderr = os.Stderr
		fmt.Println("Clone the repository")
		if err := clone.Run(); err != nil {
			fmt.Println("Error: ", err)
			exit()
		}

		copyReport := exec.Command("cp", "code/report_template.html", ".")
		copyReport.Dir = path.Dir(e)
		copyReport.Stdout = os.Stdout
		copyReport.Stderr = os.Stderr
		fmt.Println("Copy report file")
		if err := copyReport.Run(); err != nil {
			fmt.Println("Error: ", err)
			exit()
		}

		// Deploy
		//"--allow-unauthenticated"
		deploy := exec.Command("gcloud", "functions", "deploy", "Commander", "--runtime", "go116", "--trigger-http", "--memory="+c.Function_memory, "--timeout="+c.Function_timeout)
		deploy.Dir = path.Dir(e) + "/code/commander"
		deploy.Stdout = os.Stdout
		deploy.Stderr = os.Stderr
		fmt.Println("Deploying function from " + deploy.Dir)
		if err := deploy.Run(); err != nil {
			fmt.Println("Error: ", err)
			exit()
		}

		fmt.Println("Cleanup directory")
		clean := os.RemoveAll(path.Dir(e) + "/code")
		if clean != nil {
			fmt.Println(clean)
			os.Exit(1)
		}

		// fmt.Println("If you are updating a function, take into consideration GCP takes some time to update to the new version")

	}

	// Get the authorization token
	token := exec.Command("gcloud", "auth", "print-identity-token")
	buf := bytes.Buffer{}
	token.Stdout = &buf
	token.Stderr = os.Stderr
	if err := token.Run(); err != nil {
		fmt.Println("Error: ", err)
		exit()
	}
	tokenString := strings.TrimSpace(buf.String())

	if c.Debug == 1 {
		fmt.Printf("The Token: %s\n", tokenString)
	}
	// Get the project id
	projectId := exec.Command("gcloud", "config", "get-value", "project")
	bufProjectId := bytes.Buffer{}
	projectId.Stdout = &bufProjectId
	projectId.Stderr = os.Stderr
	if err := projectId.Run(); err != nil {
		fmt.Println("Error: ", err)
		exit()
	}
	projectIdString := strings.TrimSpace(bufProjectId.String())
	fmt.Printf("Project id: %s\n", projectIdString)

	fmt.Println("Triggering " + fmt.Sprint(c.Clients) + " functions")
	urlPath := "https://" + c.Function_region + "-" + projectIdString + ".cloudfunctions.net/Commander?concurrency=" + c.Concurrency + "&number=" + c.Number + "&url=" + url.QueryEscape(c.Url)

	if c.Local == 1 {
		urlPath = "http://localhost:3000/?concurrency=" + c.Concurrency + "&number=" + c.Number + "&url=" + url.QueryEscape(c.Url) + "&local=1"
	}

	client := http.Client{}

	var wg sync.WaitGroup

	var finalReport Result

	reports := make([]Report, c.Clients)

	for i := 0; i < c.Clients; i++ {
		// fmt.Println("iteration: ", i)

		wg.Add(1)
		go func(i int) {

			req, err := http.NewRequest("GET", urlPath, nil)
			if err != nil {
				fmt.Println("Error: ", err)
				exit()
			}

			req.Header = http.Header{
				"Content-Type":  []string{"application/json"},
				"Authorization": []string{"Bearer " + tokenString},
			}

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
				// fmt.Println(bodyString)
				json.Unmarshal([]byte(bodyString), &reports[i])

				finalReport.BytesRead += reports[i].Result.BytesRead
				finalReport.BytesWritten += reports[i].Result.BytesWritten
				finalReport.TimeTakenSeconds += reports[i].Result.TimeTakenSeconds
				finalReport.Req1Xx += reports[i].Result.Req1Xx
				finalReport.Req2Xx += reports[i].Result.Req2Xx
				finalReport.Req3Xx += reports[i].Result.Req3Xx
				finalReport.Req4Xx += reports[i].Result.Req4Xx
				finalReport.Req5Xx += reports[i].Result.Req5Xx
				finalReport.Others += reports[i].Result.Others
				finalReport.Latency.Mean += reports[i].Result.Latency.Mean
				finalReport.Latency.Stddev += reports[i].Result.Latency.Stddev
				finalReport.Latency.Max += reports[i].Result.Latency.Max
				finalReport.Rps.Mean += reports[i].Result.Rps.Mean
				finalReport.Rps.Stddev += reports[i].Result.Rps.Stddev
				finalReport.Rps.Max += reports[i].Result.Rps.Max
				finalReport.Rps.Percentiles.Num50 += reports[i].Result.Rps.Percentiles.Num50
				finalReport.Rps.Percentiles.Num75 += reports[i].Result.Rps.Percentiles.Num75
				finalReport.Rps.Percentiles.Num90 += reports[i].Result.Rps.Percentiles.Num90
				finalReport.Rps.Percentiles.Num95 += reports[i].Result.Rps.Percentiles.Num95
				finalReport.Rps.Percentiles.Num99 += reports[i].Result.Rps.Percentiles.Num99
			}
			wg.Done()
		}(i)
	}

	// fmt.Println(reports)
	wg.Wait()
	if c.Debug == 1 {
		for i := 0; i < c.Clients; i++ {
			fmt.Println(reports[i])
		}
		fmt.Println("----------------- final report -------------------")
		fmt.Println(finalReport)
	}

	finalReport.TimeTakenSeconds = math.Round((finalReport.TimeTakenSeconds)*100) / 100
	finalReport.Latency.Mean = math.Round((finalReport.Latency.Mean/float64(c.Clients))*100) / 100
	finalReport.Latency.Stddev = math.Round((finalReport.Latency.Stddev/float64(c.Clients))*100) / 100
	finalReport.Latency.Max = math.Round((finalReport.Latency.Max/float64(c.Clients))*100) / 100
	finalReport.Rps.Mean = math.Round((finalReport.Rps.Mean/float64(c.Clients))*100) / 100
	finalReport.Rps.Stddev = math.Round((finalReport.Rps.Stddev/float64(c.Clients))*100) / 100
	finalReport.Rps.Max = math.Round((finalReport.Rps.Max/float64(c.Clients))*100) / 100
	finalReport.Rps.Percentiles.Num50 = math.Round((finalReport.Rps.Percentiles.Num50/float64(c.Clients))*100) / 100
	finalReport.Rps.Percentiles.Num75 = math.Round((finalReport.Rps.Percentiles.Num75/float64(c.Clients))*100) / 100
	finalReport.Rps.Percentiles.Num90 = math.Round((finalReport.Rps.Percentiles.Num90/float64(c.Clients))*100) / 100
	finalReport.Rps.Percentiles.Num95 = math.Round((finalReport.Rps.Percentiles.Num95/float64(c.Clients))*100) / 100
	finalReport.Rps.Percentiles.Num99 = math.Round((finalReport.Rps.Percentiles.Num99/float64(c.Clients))*100) / 100

	t, err := template.ParseFiles("report_template.html")
	if err != nil {
		fmt.Println(err)
		return
	}

	f, err := os.Create("report.html")
	if err != nil {
		fmt.Println("create file: ", err)
		return
	}

	err = t.Execute(f, finalReport)
	if err != nil {
		fmt.Println("execute: ", err)
		return
	}
	f.Close()

	e, err := os.Executable()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("\nCleanup report template")
	cleanUpReport := os.RemoveAll(path.Dir(e) + "/report_template.html")
	if cleanUpReport != nil {
		fmt.Println(cleanUpReport)
		os.Exit(1)
	}

	fmt.Println("\nCheck report.html in the root dir.")

	// TODO - upload results to bucket
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
