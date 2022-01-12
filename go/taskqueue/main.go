package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"google.golang.org/genproto/googleapis/cloud/tasks/v2"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	defaultPort = 8080
)

func init() {
	log.SetFlags(log.LstdFlags | log.LUTC)
}

func handleError(res http.ResponseWriter, message string, cause error) {
	msg := fmt.Sprintf("%v: %v", message, cause)
	log.Printf(msg)
	http.Error(res, msg, http.StatusInternalServerError)
}

// queueHandler expects to receive a request whose body is the json representation of
// a cloud task.  It will forward the request payload onto the URL set on the task synchronously.
func taskHandler(res http.ResponseWriter, req *http.Request) {
	var data []byte
	var err error

	defer req.Body.Close()
	if data, err = io.ReadAll(req.Body); err != nil {
		handleError(res, "failed to read request body", err)
		return
	}

	taskReq := tasks.CreateTaskRequest{
		Task: &tasks.Task{
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{},
			},
		},
	}
	if err = json.Unmarshal(data, &taskReq); err != nil {
		handleError(res, "failed to generate task from request", err)
		return
	}

	taskHttp := taskReq.GetTask().GetHttpRequest()

	log.Printf("Forwarding message to %v", taskHttp.Url)

	cli := http.Client{Timeout: 3 * time.Second}

	_, err = cli.Post(taskHttp.Url, "application/json", bytes.NewReader(taskHttp.Body))
	if err != nil {
		handleError(res, "failed to send task", err)
	}

	return
}

func main() {
	var port int
	var err error
	if port, err = strconv.Atoi(os.Getenv("PORT")); err != nil {
		port = defaultPort
	}
	log.Printf("Starting service on port %d\n", port)
	http.HandleFunc("/", taskHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
