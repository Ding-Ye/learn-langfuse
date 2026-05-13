// Demo: start the ingestion server on a random port, POST a 4-event
// batch (trace + 2 observations + 1 score), then GET the trace back.
//
// Run from this directory:
//
//	go run ./cmd/demo
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	ingest "github.com/Ding-Ye/learn-langfuse/s01-trace-ingestion"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Handler: (&ingest.Server{Store: ingest.NewMemStore()}).Routes()}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Shutdown(context.Background())

	base := "http://" + ln.Addr().String()
	fmt.Println("ingest server:", base)

	now := time.Now().UTC()
	batch := ingest.BatchRequest{Batch: []ingest.Event{
		{ID: "e1", Type: ingest.TypeTraceCreate, Timestamp: now,
			Body: ingest.Body{TraceID: "demo-trace", Name: "answer-a-question", UserID: "u1"}},
		{ID: "e2", Type: ingest.TypeObservationCreate, Timestamp: now,
			Body: ingest.Body{
				TraceID: "demo-trace", ObservationID: "retrieve",
				Kind: ingest.KindSpan, Name: "retrieve-docs",
				StartTime: now, EndTime: now.Add(120 * time.Millisecond),
			}},
		{ID: "e3", Type: ingest.TypeObservationCreate, Timestamp: now.Add(time.Millisecond),
			Body: ingest.Body{
				TraceID: "demo-trace", ObservationID: "answer", ParentObservation: "retrieve",
				Kind: ingest.KindGeneration, Model: "gpt-4o-mini",
				StartTime: now.Add(120 * time.Millisecond), EndTime: now.Add(820 * time.Millisecond),
			}},
		{ID: "e4", Type: ingest.TypeScoreCreate, Timestamp: now.Add(time.Second),
			Body: ingest.Body{TraceID: "demo-trace", ScoreName: "helpfulness", ScoreValue: 0.9}},
	}}

	body, _ := json.Marshal(batch)
	req, _ := http.NewRequest("POST", base+"/api/public/ingestion", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	fmt.Printf("ingest: %d %s\n", resp.StatusCode, out)

	resp, err = http.Get(base + "/api/public/traces/demo-trace")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ = io.ReadAll(resp.Body)
	fmt.Printf("get trace: %d\n", resp.StatusCode)
	var pretty bytes.Buffer
	_ = json.Indent(&pretty, out, "", "  ")
	fmt.Println(pretty.String())
}
