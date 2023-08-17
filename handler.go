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
			if strings.ToLower(key) == "traceheader" {
				if len(values) != 1 {
					log.Printf("Error: traceheader has %d values", len(values))
					continue
				}
				traceheader = values[0]
				log.Printf("traceheader: %s", traceheader)
			}
			if strings.ToLower(key) == "tracestate" {
				if len(values) != 1 {
					log.Printf("Error: tracestate has %d values", len(values))
					continue
				}
				tracestate = values[0]
				log.Printf("tracestate: %s", tracestate)
			}
		}
		var req *http.Request
		var resp *http.Response

		reqURL := "http://" + remote + r.RequestURI
		req, err = http.NewRequest(r.Method, reqURL, r.Body)
		if err != nil {
			log.Printf("Error: could not create request: %s", err)
		} else {
			resp, err = client.Do(req)
			if err != nil {
				log.Printf("Error: could not send request: %s", err)
			} else {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Printf("Error: could not read response body %v", err)
				} else {
					log.Printf("Forwarding %s %s %d bytes", r.Method, reqURL, len(body))
					w.Write(body)
				}
			}
		}
	}
}
