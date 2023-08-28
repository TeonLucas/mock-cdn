package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Trace struct {
	Common TraceCommon `json:"common"`
	Spans  TraceSpans  `json:"spans"`
}

type Attributes map[string]interface{}

type TraceCommon struct {
	Attributes Attributes `json:"attributes"`
}

type TraceSpans []Span

type Span struct {
	Id         string     `json:"id"`
	TraceId    string     `json:"trace.id"`
	Timestamp  int64      `json:"timestamp"`
	Attributes Attributes `json:"attributes"`
}

type ApiClient struct {
	Client      *http.Client
	Headers     []string
	Url         string
	POA         string
	Account     string
	ServiceName string
}

// Make API request with error retry
func retryQuery(client *http.Client, method, url, data string, headers []string) (b []byte) {
	var res *http.Response
	var err error
	var body io.Reader

	if len(data) > 0 {
		body = strings.NewReader(data)
	}

	// up to 3 retries on API error
	for j := 1; j <= 3; j++ {
		req, _ := http.NewRequest(method, url, body)
		for _, h := range headers {
			params := strings.Split(h, ":")
			req.Header.Set(params[0], params[1])
		}
		res, err = client.Do(req)
		if err != nil {
			log.Println(err)
		}
		if res != nil {
			if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusAccepted {
				break
			}
			log.Printf("Retry %d: http status %d", j, res.StatusCode)
		} else {
			log.Printf("Retry %d: no response", j)
		}
		time.Sleep(500 * time.Millisecond)
	}
	b, err = io.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return
	}
	res.Body.Close()
	return
}

func makeTrace(id, traceId, parentId, traceParent, traceState, name, serviceName, url, method string, status int, duration, ts int64) (traces []Trace) {
	var trace Trace
	var span Span

	trace.Common.Attributes = make(Attributes)
	trace.Common.Attributes["service.name"] = serviceName
	trace.Common.Attributes["tags.serviceType"] = "OpenTelemetry"
	trace.Common.Attributes["hostname"] = "ip-172-31-34-242.us-east-2.compute.internal"

	span.Attributes = make(Attributes)
	span.Id = id
	span.TraceId = traceId
	span.Timestamp = ts
	span.Attributes["name"] = name
	span.Attributes["duration.ms"] = duration
	span.Attributes["parent.id"] = parentId
	span.Attributes["traceparent"] = traceParent
	span.Attributes["tracestate"] = traceState
	span.Attributes["http.url"] = url
	span.Attributes["http.method"] = method
	span.Attributes["http.statusCode"] = status

	//span.Attributes["error.message"] = "No error"

	trace.Spans = append(trace.Spans, span)
	traces = append(traces, trace)
	return
}

func makeClient(licenseKey, url, poa, account, serviceName string) (apiClient ApiClient) {
	apiClient.Client = http.DefaultClient
	apiClient.Headers = []string{"Content-Type:application/json", "Api-Key:" + licenseKey,
		"Data-Format:newrelic", "Data-Format-Version: 1"}
	apiClient.Url = url
	apiClient.POA = poa
	apiClient.Account = account
	apiClient.ServiceName = serviceName
	return
}

func (apiClient ApiClient) sendTraces(traces []Trace) {
	var j []byte
	var err error

	j, err = json.Marshal(traces)
	if err != nil {
		log.Printf("Error creating trace payload: %v", err)
	}

	//log.Printf("DEBUG trace payload: %s", j)

	_ = retryQuery(apiClient.Client, "POST", apiClient.Url, string(j), apiClient.Headers)
	log.Printf("Submitted OTel trace to %s: %s", apiClient.Url, j)
}

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

var reTraceParent = regexp.MustCompile(`\d{2}-(\w+)-(\w+)-\d{2}`)

func parseTraceParent(traceParent string) (traceId, parentId string) {
	matches := reTraceParent.FindStringSubmatch(traceParent)
	if len(matches) == 3 {
		traceId = matches[1]
		parentId = matches[2]
	}
	return
}

func makeNewContext(traceParent, poa, account, timestamp string) (traceId, spanId, parentId, newTraceParent, newTraceState string) {
	var err error

	// Generate spanId
	spanId, err = randomHex(8)
	if err != nil {
		log.Println("Error making new span id")
		return
	}

	if len(traceParent) > 0 {
		// Parse traceparent
		traceId, parentId = parseTraceParent(traceParent)
		log.Printf("Received parent id %s", parentId)
	} else {
		// Generate traceId
		traceId, err = randomHex(16)
		if err != nil {
			log.Println("Error making trace id")
		}
		// Use this span as the parent, since there was no caller sending context
		parentId = spanId
	}

	newTraceParent = "00-" + traceId + "-" + spanId + "-01"
	newTraceState = poa + "@nr=0-0-" + account + "-0-" + spanId + "--1--" + timestamp
	return
}
