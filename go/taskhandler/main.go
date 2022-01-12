package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

const (
	defaultPort = 8080
)

func init() {
	log.SetFlags(log.LstdFlags | log.LUTC)
}

// echoHandler simply reads the request body, logs it, and returns to sender
func echoHandler(res http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	var data []byte
	var err error
	if data, err = io.ReadAll(req.Body); err != nil {
		http.Error(res, fmt.Sprintf("failed to read request: %v", err), http.StatusBadRequest)
		return
	}
	if _, err = res.Write(data); err != nil {
		http.Error(res, fmt.Sprintf("failed to write response: %v", err), http.StatusInternalServerError)
	}
	log.Printf("%v", string(data))
	return
}

func main() {
	var port int
	var err error
	if port, err = strconv.Atoi(os.Getenv("PORT")); err != nil {
		port = defaultPort
	}
	log.Printf("Starting service on port %d\n", port)
	http.HandleFunc("/", echoHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
