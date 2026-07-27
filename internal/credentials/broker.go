package credentials

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var ErrInvalidLease = errors.New("credential lease is invalid")

type Request struct {
	TaskID     string
	ToolID     string
	Permission string
	TTL        time.Duration
}

type Lease struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"taskId"`
	ToolID     string    `json:"toolId"`
	Permission string    `json:"permission"`
	SecretRef  string    `json:"secretRef"`
	IssuedAt   time.Time `json:"issuedAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	Revoked    bool      `json:"revoked"`
}

type Broker interface {
	Issue(context.Context, Request) (Lease, error)
	Validate(context.Context, string, string, string) (Lease, error)
	Revoke(context.Context, string) error
}

type MemoryBroker struct {
	mu     sync.RWMutex
	leases map[string]Lease
	now    func() time.Time
}

func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{leases: make(map[string]Lease), now: time.Now}
}

func (b *MemoryBroker) Issue(_ context.Context, request Request) (Lease, error) {
	if request.TaskID == "" || request.ToolID == "" || request.Permission == "" {
		return Lease{}, ErrInvalidLease
	}
	if request.TTL <= 0 || request.TTL > 15*time.Minute {
		request.TTL = 5 * time.Minute
	}
	now := b.now().UTC()
	id, err := randomID()
	if err != nil {
		return Lease{}, err
	}
	lease := Lease{
		ID:         id,
		TaskID:     request.TaskID,
		ToolID:     request.ToolID,
		Permission: request.Permission,
		SecretRef:  "lease://" + id,
		IssuedAt:   now,
		ExpiresAt:  now.Add(request.TTL),
	}
	b.mu.Lock()
	b.leases[id] = lease
	b.mu.Unlock()
	return lease, nil
}

func (b *MemoryBroker) Validate(_ context.Context, leaseID, taskID, toolID string) (Lease, error) {
	b.mu.RLock()
	lease, ok := b.leases[leaseID]
	b.mu.RUnlock()
	if !ok || lease.Revoked || lease.TaskID != taskID || lease.ToolID != toolID || !lease.ExpiresAt.After(b.now()) {
		return Lease{}, ErrInvalidLease
	}
	return lease, nil
}

func (b *MemoryBroker) Revoke(_ context.Context, leaseID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	lease, ok := b.leases[leaseID]
	if !ok {
		return ErrInvalidLease
	}
	lease.Revoked = true
	b.leases[leaseID] = lease
	return nil
}

func randomID() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(data[:]), nil
}
