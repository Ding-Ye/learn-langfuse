// Package ingest implements the smallest interesting subset of langfuse's
// trace ingestion endpoint.
//
// Upstream reference: web/src/pages/api/public/ingestion.ts and
// packages/shared/src/server/ingestion/processEventBatch.ts at SHA
// 1e3535e126fc918781f45753706ff0f576b12175.
//
// What we keep in s01:
//   - the typed event envelope (`Event` with a discriminated `Type` and a
//     `Body` payload), mirroring upstream's `BaseEventBody` + discriminator.
//   - the *batch* shape — every ingest request is a list of events, and a
//     valid request produces a `BatchResponse` with per-event success.
//   - a `Store` interface + in-memory implementation, so the HTTP layer
//     doesn't bake in ClickHouse.
//
// What we cut: auth, rate-limits, S3 cache, async queue dispatch. Each
// gets its own chapter (s03/s04 auth+rate-limit, s05 queue, s06 blob).
package ingest

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// EventType is the discriminator. Upstream defines a richer set
// (trace-create, span-create, span-update, observation-create, score-create,
// event-create, sdk-log…). s01 keeps just the three that produce a
// minimum useful trace shape.
type EventType string

const (
	TypeTraceCreate       EventType = "trace-create"
	TypeObservationCreate EventType = "observation-create"
	TypeScoreCreate       EventType = "score-create"
)

// ObservationKind is the secondary discriminator for observation events.
// Mirrors upstream's `ObservationType` (SPAN | GENERATION | EVENT).
type ObservationKind string

const (
	KindSpan       ObservationKind = "SPAN"
	KindGeneration ObservationKind = "GENERATION"
	KindEvent      ObservationKind = "EVENT"
)

// Event is one entry in an ingest batch. Upstream uses pydantic +
// Zod-style discriminated unions; we use a struct with a `Type` tag and
// a typed `Body`.
type Event struct {
	ID        string    `json:"id"`        // SDK-generated dedup key
	Type      EventType `json:"type"`      // discriminator
	Timestamp time.Time `json:"timestamp"` // when the SDK saw it
	Body      Body      `json:"body"`      // discriminated payload
}

// Body holds any of the per-type payloads. The JSON wire shape is a
// single object with optional fields; we keep them all here and validate
// later. (Upstream Zod does the same: one big schema that gets refined
// at runtime by `Type`.)
type Body struct {
	// Common to traces & observations.
	ProjectID string `json:"project_id,omitempty"`
	TraceID   string `json:"trace_id"`

	// Trace-specific.
	Name      string `json:"name,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`

	// Observation-specific.
	ObservationID    string          `json:"observation_id,omitempty"`
	ParentObservation string         `json:"parent_observation_id,omitempty"`
	Kind             ObservationKind `json:"kind,omitempty"`
	Model            string          `json:"model,omitempty"`
	StartTime        time.Time       `json:"start_time,omitempty"`
	EndTime          time.Time       `json:"end_time,omitempty"`

	// Score-specific.
	ScoreName  string  `json:"score_name,omitempty"`
	ScoreValue float64 `json:"score_value,omitempty"`
}

// Validate enforces the *minimum* contract that lets later chapters
// assume the input is well-shaped. The matrix mirrors upstream's
// per-discriminator Zod refinements.
func (e Event) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return errors.New("event.id is required")
	}
	if e.Body.TraceID == "" {
		return errors.New("body.trace_id is required")
	}
	switch e.Type {
	case TypeTraceCreate:
		if e.Body.Name == "" {
			return errors.New("trace-create: body.name is required")
		}
	case TypeObservationCreate:
		if e.Body.ObservationID == "" {
			return errors.New("observation-create: body.observation_id is required")
		}
		switch e.Body.Kind {
		case KindSpan, KindGeneration, KindEvent:
		default:
			return fmt.Errorf("observation-create: invalid kind %q", e.Body.Kind)
		}
		if e.Body.StartTime.IsZero() {
			return errors.New("observation-create: body.start_time is required")
		}
	case TypeScoreCreate:
		if e.Body.ScoreName == "" {
			return errors.New("score-create: body.score_name is required")
		}
	default:
		return fmt.Errorf("unknown event.type %q", e.Type)
	}
	return nil
}

// BatchRequest is what the SDK POSTs.
type BatchRequest struct {
	Batch []Event `json:"batch"`
}

// BatchResponse mirrors upstream's per-event ack list. `Successes` and
// `Errors` are returned regardless of partial failure — clients can
// retry only the failed `id`s.
type BatchResponse struct {
	Successes []EventAck    `json:"successes"`
	Errors    []EventReject `json:"errors"`
}

type EventAck struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
}

type EventReject struct {
	ID      string `json:"id"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}
