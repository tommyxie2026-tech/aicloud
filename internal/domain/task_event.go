package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidTaskEvent         = errors.New("invalid task event")
	ErrInvalidOutboxMessage     = errors.New("invalid outbox message")
	ErrInvalidIdempotencyRecord = errors.New("invalid idempotency record")
)

type TaskEventActor struct {
	PrincipalType string `json:"principalType"`
	SubjectID     string `json:"subjectId"`
}

type TaskEvent struct {
	EventID       string          `json:"eventId"`
	TenantID      string          `json:"tenantId"`
	ProjectID     string          `json:"projectId"`
	TaskID        string          `json:"taskId"`
	Sequence      int64           `json:"sequence"`
	EventType     string          `json:"eventType"`
	Actor         TaskEventActor  `json:"actor"`
	Payload       json.RawMessage `json:"payload"`
	RequestID     string          `json:"requestId,omitempty"`
	TraceID       string          `json:"traceId"`
	SchemaVersion int             `json:"schemaVersion"`
	OccurredAt    time.Time       `json:"occurredAt"`
	CreatedAt     time.Time       `json:"createdAt"`
}

func (e TaskEvent) Validate() error {
	if strings.TrimSpace(e.EventID) == "" || strings.TrimSpace(e.TenantID) == "" ||
		strings.TrimSpace(e.ProjectID) == "" || strings.TrimSpace(e.TaskID) == "" ||
		strings.TrimSpace(e.EventType) == "" || strings.TrimSpace(e.TraceID) == "" {
		return ErrInvalidTaskEvent
	}
	if e.Sequence < 1 || e.SchemaVersion < 1 || e.OccurredAt.IsZero() || e.CreatedAt.IsZero() {
		return ErrInvalidTaskEvent
	}
	if strings.TrimSpace(e.Actor.PrincipalType) == "" || strings.TrimSpace(e.Actor.SubjectID) == "" {
		return ErrInvalidTaskEvent
	}
	if len(e.Payload) == 0 || !json.Valid(e.Payload) {
		return fmt.Errorf("%w: payload must be valid JSON", ErrInvalidTaskEvent)
	}
	return nil
}

func CanonicalTaskStateEvent(status TaskStatus) (string, bool) {
	switch status {
	case TaskCreated:
		return "TaskCreated", true
	case TaskPlanning:
		return "TaskPlanningStarted", true
	case TaskRouting:
		return "TaskRoutingStarted", true
	case TaskExecuting:
		return "TaskExecutionStarted", true
	case TaskWaitingApproval:
		return "TaskApprovalRequested", true
	case TaskValidating:
		return "TaskValidationStarted", true
	case TaskCompleted:
		return "TaskCompleted", true
	case TaskFailed:
		return "TaskFailed", true
	case TaskCancelled:
		return "TaskCancelled", true
	case TaskExpired:
		return "TaskExpired", true
	default:
		return "", false
	}
}

type OutboxStatus string

const (
	OutboxPending    OutboxStatus = "pending"
	OutboxDelivering OutboxStatus = "delivering"
	OutboxDelivered  OutboxStatus = "delivered"
	OutboxDeadLetter OutboxStatus = "dead_letter"
)

type OutboxMessage struct {
	OutboxID       string          `json:"outboxId"`
	TenantID       string          `json:"tenantId"`
	ProjectID      string          `json:"projectId"`
	TaskID         string          `json:"taskId,omitempty"`
	AggregateType  string          `json:"aggregateType"`
	AggregateID    string          `json:"aggregateId"`
	EventType      string          `json:"eventType"`
	Payload        json.RawMessage `json:"payload"`
	Destination    string          `json:"destination"`
	IdempotencyKey string          `json:"idempotencyKey"`
	Status         OutboxStatus    `json:"status"`
	Attempts       int             `json:"attempts"`
	AvailableAt    time.Time       `json:"availableAt"`
	CreatedAt      time.Time       `json:"createdAt"`
	DeliveredAt    *time.Time      `json:"deliveredAt,omitempty"`
}

func (m OutboxMessage) Validate() error {
	if strings.TrimSpace(m.OutboxID) == "" || strings.TrimSpace(m.TenantID) == "" ||
		strings.TrimSpace(m.ProjectID) == "" || strings.TrimSpace(m.AggregateType) == "" ||
		strings.TrimSpace(m.AggregateID) == "" || strings.TrimSpace(m.EventType) == "" ||
		strings.TrimSpace(m.Destination) == "" || strings.TrimSpace(m.IdempotencyKey) == "" {
		return ErrInvalidOutboxMessage
	}
	if m.Attempts < 0 || m.AvailableAt.IsZero() || m.CreatedAt.IsZero() {
		return ErrInvalidOutboxMessage
	}
	switch m.Status {
	case OutboxPending, OutboxDelivering, OutboxDelivered, OutboxDeadLetter:
	default:
		return ErrInvalidOutboxMessage
	}
	if len(m.Payload) == 0 || !json.Valid(m.Payload) {
		return fmt.Errorf("%w: payload must be valid JSON", ErrInvalidOutboxMessage)
	}
	return nil
}

type IdempotencyStatus string

const (
	IdempotencyInProgress      IdempotencyStatus = "in_progress"
	IdempotencyCompleted       IdempotencyStatus = "completed"
	IdempotencyFailedRetryable IdempotencyStatus = "failed_retryable"
	IdempotencyFailedFinal     IdempotencyStatus = "failed_final"
)

type IdempotencyRecord struct {
	TenantID        string            `json:"tenantId"`
	ProjectID       string            `json:"projectId"`
	Operation       string            `json:"operation"`
	Key             string            `json:"key"`
	RequestDigest   string            `json:"requestDigest"`
	Status          IdempotencyStatus `json:"status"`
	ResourceID      string            `json:"resourceId,omitempty"`
	ResponseCode    int               `json:"responseCode,omitempty"`
	ResponseDigest  string            `json:"responseDigest,omitempty"`
	ResponsePayload json.RawMessage   `json:"responsePayload,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	ExpiresAt       time.Time         `json:"expiresAt"`
}

func (r IdempotencyRecord) Validate() error {
	if strings.TrimSpace(r.TenantID) == "" || strings.TrimSpace(r.ProjectID) == "" ||
		strings.TrimSpace(r.Operation) == "" || strings.TrimSpace(r.Key) == "" ||
		strings.TrimSpace(r.RequestDigest) == "" || r.CreatedAt.IsZero() || r.ExpiresAt.IsZero() {
		return ErrInvalidIdempotencyRecord
	}
	if !r.ExpiresAt.After(r.CreatedAt) {
		return fmt.Errorf("%w: expires_at must be after created_at", ErrInvalidIdempotencyRecord)
	}
	switch r.Status {
	case IdempotencyInProgress, IdempotencyCompleted, IdempotencyFailedRetryable, IdempotencyFailedFinal:
	default:
		return ErrInvalidIdempotencyRecord
	}
	if len(r.ResponsePayload) > 0 && !json.Valid(r.ResponsePayload) {
		return fmt.Errorf("%w: response payload must be valid JSON", ErrInvalidIdempotencyRecord)
	}
	return nil
}
