# Cloud Tasks CSV Example

## Overview

This project provides an example of implementing per-row processing using Google Cloud Tasks in go, but without
actually using Cloud Tasks.

## Requirements

- Git
- Docker

## Running The Project

Use the following commands to run the project:

```shell
% git clone github.com/mfinding/gcp-task-example <yourdir>

% cd <yourdir>/go

% docker compose build && docker compose up -d

% curl "http://localhost:8081/?f=/tmp/example.csv"
```

## Explanation

There are 3 parts to this example:

1. File Handler
2. Task Queue
3. Task Handler

### File Handler

The file handler expects to receive a message from an http client (e.g. curl/postman) which contains a
single query parameter "f".  This parameter contains the path to a file that it should process and the
path MUST be accessible within the container.

This is to emulate the scenario where a file is being uploaded and then a service is being notified it
should process a file.  In practice, it would read the file from a cloud bucket somewhere (example of this
pending).

Assuming it can find the file it will read every line and dispatch a "Cloud Task" to the local simulator,
the Task Queue.  It will log any row it reads and any error it encounters.

###  Task Queue

The task queue emulates cloud tasks in that it expects to receive a GCP Cloud Task json message to process.
That's where the similarity ends, it is a synchronous request that will immediately read the task, extract
the URL and request body from the Cloud Task, and send said body to said URL.

### Task Handler

Final recipient of the original row that is expected to do processing of it.  In the real world this is likely a
database update of some sort, in this example it just logs the original message sent. 

All-in-all a bunch of compute and network hops to print a message :-)

## TODO

Finish off the example that actually uses Cloud Tasks (where GCP_PROJECT is set).

## Disclaimers

This is not production software!  It is a quick example to show the principle behind taking a file event which
provides a location and reading that file.  No method/path/param checking, minimal error checking, lots of 
repetition, no file concurrency, etc, etc.

Buyer beware....