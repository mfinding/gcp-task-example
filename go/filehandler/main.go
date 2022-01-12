package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"fmt"
	"google.golang.org/genproto/googleapis/cloud/tasks/v2"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
	"io"
	"log"
	"os"
)

func init() {
	log.SetFlags(log.LstdFlags | log.LUTC)
}

const (
	defaultPort       = 8080
	defaultQueueUrl   = "http://localhost:8082"
	defaultHandlerUrl = "http://localhost:8083"
	defaultRegion     = "europe-west1"
	defaultQueue      = "csv-queue"
	gcpQueueFormat    = "projects/%s/locations/%s/queues/%s"

	queueEnv      = "TASK_QUEUE_URL" // Only used for local testing
	handlerEnv    = "TASK_HANDLER_URL"
	gcpProjectEnv = "GCP_PROJECT"
	gcpRegionEnv  = "GCP_REGION"
	gcpQueueEnv   = "GCP_TASK_QUEUE"
)

type gcpConfig struct {
	project   string
	region    string
	taskQueue string
}

func (cfg gcpConfig) queuePath() string {
	return fmt.Sprintf(gcpQueueFormat, cfg.project, cfg.region, cfg.taskQueue)
}

var (
	inputFile  string
	taskCli    *cloudtasks.Client
	useGcp     bool
	gcsPattern = regexp.MustCompile("^gs://.*")

	handlerUrl = defaultHandlerUrl
	queueUrl   = defaultQueueUrl

	gcpCfg = gcpConfig{
		region:    defaultRegion,
		taskQueue: defaultQueue,
	}
)

func init() {
	var err error

	// Read in all environment variables
	for env, v := range map[string]*string{
		queueEnv:      &queueUrl,
		handlerEnv:    &handlerUrl,
		gcpProjectEnv: &gcpCfg.project,
		gcpRegionEnv:  &gcpCfg.region,
		gcpQueueEnv:   &gcpCfg.taskQueue,
	} {
		if envVal := os.Getenv(env); envVal != "" {
			*v = envVal
		}
	}

	if gcpCfg.project != "" {
		if taskCli, err = cloudtasks.NewClient(context.Background()); err != nil {
			log.Fatalf("failed to create cloud tasks client")
		}
	}
}

func min(x int, y int) int {
	if x <= y {
		return x
	}
	return y
}

func queueTask(rowData map[string]string) error {
	var err error
	req := &tasks.CreateTaskRequest{
		Parent: gcpQueueFormat,
		Task: &tasks.Task{
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        handlerUrl,
				},
			},
		},
		ResponseView: 0,
	}

	// Marshal indent purely for this example - don't indent in real system->system comms
	data, err := json.MarshalIndent(rowData, "", "  ")

	if err != nil {
		return fmt.Errorf("failed to encode row: %v", err)
	}
	log.Println(string(data))

	req.Task.GetHttpRequest().Body = data

	// If running in GCP we'll be using protobufs but locally we use json
	if useGcp {
		_, err = taskCli.CreateTask(context.Background(), req)
		if err != nil {
			return fmt.Errorf("failed to queue row: %w", err)
		}
		return nil
	}

	// Running locally
	data, err = json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to build local task request: %w", err)
	}
	log.Printf("Sending task %v to %v", string(data), queueUrl)
	// Don't use the default client, no timeout
	cli := http.Client{
		Timeout: time.Second * 3,
	}
	resp, err := cli.Post(queueUrl, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("faild to send local task: %w", err)
	}
	if resp.StatusCode > 299 {
		return fmt.Errorf("task schedule failed: %v", resp.Status)
	}

	return nil
}

func processGCSFile(res http.ResponseWriter, filename string) error {
	http.Error(res, "Not Implemented", http.StatusBadRequest)
	return nil
}

func processLocalFile(res http.ResponseWriter, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return err
	}
	defer f.Close()
	return processFile(f)
}

func processFile(reader io.Reader) error {
	csvIn := csv.NewReader(bufio.NewReader(reader))

	header, err := csvIn.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("input file %v is empty", inputFile)
		}
		return fmt.Errorf("failed to read header from input file %v", inputFile)
	}

	line := 1

	for {
		line++

		var row []string
		row, err = csvIn.Read()

		if errors.Is(err, io.EOF) {
			break
		} else if errors.Is(err, csv.ErrFieldCount) {
			log.Printf("Warning: Unexpected number of fields on line %v", line)
			// Crack on and let the downstream validate the missing fields
		} else if err != nil {
			return fmt.Errorf("failed to process line %v: %v", line, err)
		}

		// Map of headers to the fields in the row, will be JSON marshaled as { "<header>": "<row value>" }
		rowData := make(map[string]string, len(header))

		end := min(len(row), len(header))

		for i := 0; i < end; i++ {
			rowData[header[i]] = row[i]
		}

		if err = queueTask(rowData); err != nil {
			return fmt.Errorf("failed to queue task on row %v: %v", line, err)
		}
	}
	return nil
}

func handler(res http.ResponseWriter, req *http.Request) {
	files, ok := req.URL.Query()["f"]
	if !ok || len(files) == 0 {
		http.Error(res, "parameter f (file) is mandatory", http.StatusBadRequest)
		return
	}

	var err error

	if gcsPattern.MatchString(files[0]) {
		if !useGcp {
			http.Error(res, "GCP is not configured", http.StatusBadRequest)
			return
		}
		err = processGCSFile(res, files[0])
	} else {
		err = processLocalFile(res, files[0])
	}

	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	var port int
	var err error
	if port, err = strconv.Atoi(os.Getenv("PORT")); err != nil {
		port = defaultPort
	}
	log.Printf("Starting service on port %d\n", port)
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
