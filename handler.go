package main

import (
	"io"
	"log"
	"net/http"
	"strings"
)

func makeHandleAll(remote string) func(w http.ResponseWriter, r *http.Request) {

	client := http.DefaultClient

	return func(w http.ResponseWriter, r *http.Request) {
		var traceheader, tracestate string
		var err error

		log.Printf("%s %s %s", r.Method, r.Host, r.RequestURI)
		defer r.Body.Close()

		for key, values := range r.Header {
			if strings.ToLower(key) == "accept" ||
				strings.ToLower(key) == "user-agent" || strings.ToLower(key) == "content-length" {
				log.Printf("%s: %s", key, strings.Join(values, "; "))
			}
			if strings.Contains(strings.ToLower(key), "trace") {
				if len(values) != 1 {
					log.Printf("Error: %s has %d values", key, len(values))
					continue
				}
				if strings.Contains(strings.ToLower(key), "header") {
					traceheader = values[0]
					log.Printf("%s: %s", key, traceheader)
				} else {
					tracestate = values[0]
					log.Printf("%s: %s", key, tracestate)
				}
			}
		}
		var req *http.Request
		var resp *http.Response
		var body []byte

		reqURL := "http://" + remote + r.RequestURI
		req, err = http.NewRequest(r.Method, reqURL, r.Body)
		if err != nil {
			log.Printf("Error: could not create remote request: %s", err)
		} else {
			resp, err = client.Do(req)
			if err != nil {
				log.Printf("Error: could not send remote request: %s", err)
			} else {
				body, err = io.ReadAll(resp.Body)
				if err != nil {
					log.Printf("Error: could not read remote response body %v", err)
				} else {
					log.Printf("Forwarding %s %s %d bytes", r.Method, reqURL, len(body))
					_, err = w.Write(body)
					if err != nil {
						log.Printf("Error: could not write local with remote response (%d bytes) %v", len(body), err)
					}
				}
			}
		}
	}
}
