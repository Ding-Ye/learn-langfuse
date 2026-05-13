package ingest

import (
	"errors"
	"sort"
	"sync"
	"time"
)

func nsOf(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

// Store is the interface every chapter beyond s01 talks to. s01 ships
// only the in-memory implementation; s07 will swap in a column-oriented
// writer that mirrors upstream's ClickHouse client.
type Store interface {
	UpsertTrace(t Trace) error
	UpsertObservation(o Observation) error
	UpsertScore(s Score) error

	GetTrace(traceID string) (Trace, bool)
	ListObservations(traceID string) []Observation
	ListScores(traceID string) []Score
}

// Trace is the materialised root row. Upstream's web/server projects the
// same shape out of ClickHouse for the UI's traces table.
type Trace struct {
	ID        string
	ProjectID string
	Name      string
	UserID    string
	SessionID string
}

// Observation is the (span | generation | event) row. ParentObservation
// builds a tree inside a single Trace.
//
// StartNS / EndNS are kept on their own pair of fields so the
// in-memory store can derive trace duration without re-decoding bodies.
type Observation struct {
	ID                  string
	TraceID             string
	ParentObservationID string
	Kind                ObservationKind
	Name                string
	Model               string
	StartNS             int64 // unix-nano; 0 if unset
	EndNS               int64 // unix-nano; 0 if unset
}

// Score is the eval-style score row.
type Score struct {
	TraceID         string
	ObservationID   string // optional — score may attach to whole trace
	Name            string
	Value           float64
}

// MemStore is the in-memory implementation: three maps keyed by id.
// Safe for concurrent use by the HTTP handler under sync.RWMutex.
type MemStore struct {
	mu     sync.RWMutex
	traces map[string]Trace
	obs    map[string]map[string]Observation // traceID -> obsID -> Observation
	scores map[string][]Score                // traceID -> []Score
}

// NewMemStore returns an empty MemStore.
func NewMemStore() *MemStore {
	return &MemStore{
		traces: make(map[string]Trace),
		obs:    make(map[string]map[string]Observation),
		scores: make(map[string][]Score),
	}
}

func (s *MemStore) UpsertTrace(t Trace) error {
	if t.ID == "" {
		return errors.New("trace.id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.traces[t.ID] = t
	return nil
}

func (s *MemStore) UpsertObservation(o Observation) error {
	if o.ID == "" || o.TraceID == "" {
		return errors.New("observation.id and trace_id are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	bucket, ok := s.obs[o.TraceID]
	if !ok {
		bucket = make(map[string]Observation)
		s.obs[o.TraceID] = bucket
	}
	bucket[o.ID] = o
	return nil
}

func (s *MemStore) UpsertScore(sc Score) error {
	if sc.TraceID == "" || sc.Name == "" {
		return errors.New("score.trace_id and score.name are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scores[sc.TraceID] = append(s.scores[sc.TraceID], sc)
	return nil
}

func (s *MemStore) GetTrace(traceID string) (Trace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.traces[traceID]
	return t, ok
}

func (s *MemStore) ListObservations(traceID string) []Observation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bucket := s.obs[traceID]
	out := make([]Observation, 0, len(bucket))
	for _, o := range bucket {
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartNS < out[j].StartNS })
	return out
}

func (s *MemStore) ListScores(traceID string) []Score {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Score, len(s.scores[traceID]))
	copy(out, s.scores[traceID])
	return out
}

// Apply consumes a validated Event and writes the appropriate row(s) to
// the store. This is the seam upstream's `processEventBatch` opens at —
// in s01 we run it synchronously; s05 introduces a queue.
func Apply(s Store, e Event) error {
	if err := e.Validate(); err != nil {
		return err
	}
	switch e.Type {
	case TypeTraceCreate:
		return s.UpsertTrace(Trace{
			ID:        e.Body.TraceID,
			ProjectID: e.Body.ProjectID,
			Name:      e.Body.Name,
			UserID:    e.Body.UserID,
			SessionID: e.Body.SessionID,
		})
	case TypeObservationCreate:
		return s.UpsertObservation(Observation{
			ID:                  e.Body.ObservationID,
			TraceID:             e.Body.TraceID,
			ParentObservationID: e.Body.ParentObservation,
			Kind:                e.Body.Kind,
			Name:                e.Body.Name,
			Model:               e.Body.Model,
			StartNS: nsOf(e.Body.StartTime),
			EndNS:   nsOf(e.Body.EndTime),
		})
	case TypeScoreCreate:
		return s.UpsertScore(Score{
			TraceID:       e.Body.TraceID,
			ObservationID: e.Body.ObservationID,
			Name:          e.Body.ScoreName,
			Value:         e.Body.ScoreValue,
		})
	}
	return errors.New("apply: unreachable")
}
