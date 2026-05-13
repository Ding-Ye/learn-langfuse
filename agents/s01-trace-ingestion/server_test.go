package ingest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newSrv(t *testing.T) *httptest.Server {
	t.Helper()
	s := &Server{Store: NewMemStore()}
	return httptest.NewServer(s.Routes())
}

func post(t *testing.T, srv *httptest.Server, path string, body any) (*http.Response, []byte) {
	t.Helper()
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", srv.URL+path, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	out, _ := readAll(resp.Body)
	return resp, out
}

func readAll(r interface {
	Read(p []byte) (int, error)
}) ([]byte, error) {
	buf := new(bytes.Buffer)
	tmp := make([]byte, 4096)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
		}
		if err != nil {
			if err.Error() == "EOF" {
				return buf.Bytes(), nil
			}
			return buf.Bytes(), err
		}
	}
}

func TestIngestSingleTrace(t *testing.T) {
	srv := newSrv(t)
	defer srv.Close()

	now := time.Now().UTC()
	req := BatchRequest{Batch: []Event{
		{
			ID: "ev-1", Type: TypeTraceCreate, Timestamp: now,
			Body: Body{TraceID: "tr-1", Name: "demo"},
		},
		{
			ID: "ev-2", Type: TypeObservationCreate, Timestamp: now,
			Body: Body{
				TraceID: "tr-1", ObservationID: "ob-1",
				Kind: KindGeneration, Model: "gpt-4o-mini",
				StartTime: now, EndTime: now.Add(500 * time.Millisecond),
			},
		},
	}}

	resp, body := post(t, srv, "/api/public/ingestion", req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d body=%s", resp.StatusCode, body)
	}

	var br BatchResponse
	_ = json.Unmarshal(body, &br)
	if len(br.Successes) != 2 || len(br.Errors) != 0 {
		t.Fatalf("expected 2 successes 0 errors, got %+v", br)
	}
}

func TestIngestPartialFailure(t *testing.T) {
	srv := newSrv(t)
	defer srv.Close()

	req := BatchRequest{Batch: []Event{
		{ID: "ok", Type: TypeTraceCreate, Body: Body{TraceID: "t1", Name: "demo"}},
		{ID: "bad", Type: TypeTraceCreate, Body: Body{TraceID: "t2" /* no Name */}},
	}}

	resp, body := post(t, srv, "/api/public/ingestion", req)
	if resp.StatusCode != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d body=%s", resp.StatusCode, body)
	}
	var br BatchResponse
	_ = json.Unmarshal(body, &br)
	if len(br.Successes) != 1 || len(br.Errors) != 1 {
		t.Fatalf("got %+v", br)
	}
	if br.Errors[0].ID != "bad" || !strings.Contains(br.Errors[0].Message, "name") {
		t.Errorf("error shape: %+v", br.Errors[0])
	}
}

func TestGetTraceReturnsNestedShape(t *testing.T) {
	srv := newSrv(t)
	defer srv.Close()

	now := time.Now().UTC()
	post(t, srv, "/api/public/ingestion", BatchRequest{Batch: []Event{
		{ID: "1", Type: TypeTraceCreate, Body: Body{TraceID: "T", Name: "root"}},
		{ID: "2", Type: TypeObservationCreate, Body: Body{
			TraceID: "T", ObservationID: "O1", Kind: KindSpan, StartTime: now,
		}},
		{ID: "3", Type: TypeObservationCreate, Body: Body{
			TraceID: "T", ObservationID: "O2", ParentObservation: "O1",
			Kind: KindGeneration, Model: "gpt-4o", StartTime: now.Add(time.Millisecond),
		}},
		{ID: "4", Type: TypeScoreCreate, Body: Body{
			TraceID: "T", ScoreName: "helpfulness", ScoreValue: 0.9,
		}},
	}})

	r, err := http.Get(srv.URL + "/api/public/traces/T")
	if err != nil {
		t.Fatalf("GET trace: %v", err)
	}
	defer r.Body.Close()
	body, _ := readAll(r.Body)
	resp := r
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET trace: %d body=%s", resp.StatusCode, body)
	}

	var got struct {
		Trace        Trace
		Observations []Observation
		Scores       []Score
	}
	_ = json.Unmarshal(body, &got)
	if got.Trace.Name != "root" {
		t.Errorf("Trace.Name=%q", got.Trace.Name)
	}
	if len(got.Observations) != 2 {
		t.Errorf("Observations=%d want 2", len(got.Observations))
	}
	if got.Observations[0].StartNS > got.Observations[1].StartNS {
		t.Errorf("observations not sorted ascending by StartNS")
	}
	if len(got.Scores) != 1 || got.Scores[0].Name != "helpfulness" {
		t.Errorf("Scores=%+v", got.Scores)
	}
}

func TestRejectsNonJSON(t *testing.T) {
	srv := newSrv(t)
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/public/ingestion",
		strings.NewReader("not json"))
	req.Header.Set("Content-Type", "text/plain")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d, want 415", resp.StatusCode)
	}
}

func TestHealthz(t *testing.T) {
	srv := newSrv(t)
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/healthz")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
