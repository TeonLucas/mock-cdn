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

		// Copy headers to relayed request (except W3C trace context)
		for key, values := range r.Header {
			keyLC := strings.ToLower(key)
			if strings.Contains(keyLC, "trace") {
				if len(values) != 1 {
					log.Printf("Error: %s has %d values", key, len(values))
					continue
				}
				if strings.Contains(keyLC, "parent") {
					traceParent = values[0]
					log.Printf("Received %s: %s", key, traceParent)
				} else if strings.Contains(keyLC, "state") {
					traceState = values[0]
					log.Printf("Received %s: %s", key, traceState)
				}
			} else {
				if keyLC == "newrelic" || keyLC == "user-agent" || keyLC == "content-length" {
					log.Printf("%s: %s", key, strings.Join(values, "; "))
				}
				for _, v := range values {
					req.Header.Set(key, v)
				}
			}
		}

		// Make new trace context and set headers
		traceId, spanId, parentId, newTraceParent, newTraceState = makeNewContext(traceParent, apiClient.POA, apiClient.Account, timestamp)
		req.Header.Set("Traceparent", newTraceParent)
		req.Header.Set("Tracestate", newTraceState)

		// Send traces to NR
		go func() {
			traces := makeTrace(spanId, traceId, parentId, newTraceParent, newTraceState, r.RequestURI, apiClient.ServiceName,
				reqURL, r.Method, 200, tDuration.Milliseconds(), now.UnixMilli())
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
