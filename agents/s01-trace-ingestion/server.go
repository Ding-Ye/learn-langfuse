package ingest

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Server is the smallest interesting HTTP face of ingestion. It mirrors
// the contract of upstream's POST /api/public/ingestion (minus auth,
// rate-limits, OTel, async queueing — each gets its own chapter).
type Server struct {
	Store Store
}

// NewServer wires the routes on a fresh mux and returns it.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/public/ingestion", s.ingest)
	mux.HandleFunc("GET /api/public/traces/", s.getTrace)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

func (s *Server) ingest(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		http.Error(w, "expected application/json", http.StatusUnsupportedMediaType)
		return
	}

	var batch BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "decode batch: "+err.Error(), http.StatusBadRequest)
		return
	}

	resp := BatchResponse{}
	for _, e := range batch.Batch {
		if err := Apply(s.Store, e); err != nil {
			resp.Errors = append(resp.Errors, EventReject{
				ID: e.ID, Status: http.StatusBadRequest, Message: err.Error(),
			})
			continue
		}
		resp.Successes = append(resp.Successes, EventAck{ID: e.ID, Status: http.StatusOK})
	}

	// Upstream returns 207 Multi-Status when there's a mix of successes
	// and errors. We do the same — the SDK already understands it.
	w.Header().Set("Content-Type", "application/json")
	switch {
	case len(resp.Errors) > 0 && len(resp.Successes) > 0:
		w.WriteHeader(http.StatusMultiStatus)
	case len(resp.Errors) > 0:
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// getTrace handles GET /api/public/traces/<trace-id>. Returns the
// materialised trace shape with nested observations and scores —
// the same projection the upstream web UI's trace-detail page renders.
func (s *Server) getTrace(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/public/traces/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	t, ok := s.Store.GetTrace(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	out := struct {
		Trace        Trace         `json:"trace"`
		Observations []Observation `json:"observations"`
		Scores       []Score       `json:"scores"`
	}{
		Trace:        t,
		Observations: s.Store.ListObservations(id),
		Scores:       s.Store.ListScores(id),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
