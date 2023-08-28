package main

import (
	"log"
	"net/http"
	"os"
)

const (
	LOCAL_ADDRESS  = "localhost:8088"
	REMOTE_ADDRESS = "localhost:8080"
	TRACE_ENDPOINT = "https://trace-api.newrelic.com/trace/v1"
	SERVICE_NAME   = "Mock CDN"
)

func main() {

	account := os.Getenv("NEW_RELIC_ACCOUNT")
	if len(account) == 0 {
		log.Printf("Please set env var NEW_RELIC_ACCOUNT")
		os.Exit(0)
	}
	poa := os.Getenv("NEW_RELIC_POA")
	if len(account) == 0 {
		poa = account
	}
	licenseKey := os.Getenv("NEW_RELIC_LICENSE_KEY")
	if len(licenseKey) == 0 {
		log.Printf("Please set env var NEW_RELIC_LICENSE_KEY")
		os.Exit(0)
	}
	traceEndpoint := os.Getenv("TRACE_ENDPOINT")
	if len(traceEndpoint) == 0 {
		traceEndpoint = TRACE_ENDPOINT
	}
	serviceName := os.Getenv("SERVICE_NAME")
	if len(serviceName) == 0 {
		serviceName = SERVICE_NAME
	}
	local := os.Getenv("LOCAL_ADDRESS")
	if len(local) == 0 {
		local = LOCAL_ADDRESS
	}
	remote := os.Getenv("REMOTE_ADDRESS")
	if len(remote) == 0 {
		remote = REMOTE_ADDRESS
	}

	// HTTP client for Trace API
	traceClient := makeClient(licenseKey, traceEndpoint, poa, account, serviceName)

	// The / pattern matches everything
	http.HandleFunc("/", makeHandleAll(remote, traceClient))

	log.Printf("Local Server at %s", local)
	log.Fatal(http.ListenAndServe(local, nil))
}
