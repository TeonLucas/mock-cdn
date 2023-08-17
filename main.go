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

	// The / pattern matches everything
	http.HandleFunc("/", makeHandleAll(remote))

	addr := LOCAL_ADDRESS
	log.Printf("Server listening at %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
