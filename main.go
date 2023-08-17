package main

import (
	"log"
	"net/http"
	"os"
)

const (
	LOCAL_ADDRESS  = "localhost:8088"
	REMOTE_ADDRESS = "localhost:8080"
)

func main() {

	local := os.Getenv("LOCAL_ADDRESS")
	if len(local) == 0 {
		local = LOCAL_ADDRESS
	}
	remote := os.Getenv("REMOTE_ADDRESS")
	if len(remote) == 0 {
		remote = REMOTE_ADDRESS
	}

	licenseKey := os.Getenv("NEW_RELIC_LICENSE_KEY")
	if len(licenseKey) == 0 {
		log.Printf("Please set env var NEW_RELIC_LICENSE_KEY")
		os.Exit(0)
	}

	// The / pattern matches everything
	http.HandleFunc("/", makeHandleAll(remote))

	addr := LOCAL_ADDRESS
	log.Printf("Server listening at %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
