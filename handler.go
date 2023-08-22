package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func makeHandleAll(remote string, apiClient ApiClient) func(w http.ResponseWriter, r *http.Request) {

	client := http.DefaultClient

	return func(w http.ResponseWriter, r *http.Request) {
		var traceParent, traceState, traceId, spanId, parentId, newTraceParent, newTraceState string
		var err error

		tStart := time.Now()

		log.Printf("%s %s %s", r.Method, r.Host, r.RequestURI)
		defer r.Body.Close()

		// Initialize new remote request
		var req *http.Request
		reqURL := "http://" + remote + r.RequestURI
		log.Printf("Creating request to remote: %s %s", r.Method, reqURL)
		req, err = http.NewRequest(r.Method, reqURL, r.Body)
		if err != nil {
			log.Printf("Error: could not create remote request: %s", err)
			return
		}

		// Add some delay to mock processing time
		time.Sleep(10 * time.Millisecond)
		now := time.Now()
		timestamp := fmt.Sprintf("%d", now.UnixMilli())
		tDuration := now.Sub(tStart)

		// Copy select headers
		for key, values := range r.Header {
			if strings.ToLower(key) == "newrelic" {
				if len(values) != 1 {
					log.Printf("Error: %s has %d values", key, len(values))
					continue
				}
				log.Printf("%s: %s", key, values[0])
			} else if strings.ToLower(key) == "accept" || strings.ToLower(key) == "user-agent" ||
				strings.ToLower(key) == "content-length" {
				log.Printf("%s: %s", key, strings.Join(values, "; "))
				if len(values) > 0 {
					req.Header.Set(key, values[0])
				}
			} else if strings.Contains(strings.ToLower(key), "trace") {
				if len(values) != 1 {
					log.Printf("Error: %s has %d values", key, len(values))
					continue
				}
				if strings.Contains(strings.ToLower(key), "parent") {
					traceParent = values[0]
					log.Printf("Received %s: %s", key, traceParent)
				} else if strings.Contains(strings.ToLower(key), "state") {
					traceState = values[0]
					log.Printf("Received %s: %s", key, traceState)
				}
			} else if strings.ToLower(key) == "cookie" {
				if len(values) > 0 {
					req.Header.Set(key, values[0])
				}
			}
		}

		// Make new trace context and set headers
		traceId, spanId, parentId, newTraceParent, newTraceState = makeNewContext(traceParent, apiClient.POA, apiClient.Account, timestamp)
		req.Header.Set("Traceparent", newTraceParent)
		req.Header.Set("Tracestate", newTraceState)

		// Send traces to NR
		go func() {
			traces := makeTrace(spanId, traceId, parentId, newTraceParent, newTraceState, r.RequestURI, reqURL, r.Method, 200,
				tDuration.Milliseconds(), now.UnixMilli())
			log.Printf("Sending traceparent: %s", newTraceParent)
			log.Printf("Sending tracestate: %s", newTraceState)
			apiClient.sendTraces(traces)
		}()

		// Relay request to remote
		var resp *http.Response
		var body []byte
		resp, err = client.Do(req)
		if err != nil {
			log.Printf("Error: could not send remote request: %s", err)
			return
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
