package outbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

var ErrDestinationNotConfigured = errors.New("outbox destination is not configured")

type Store interface {
	Lease(context.Context, string, int, time.Time, time.Duration) ([]repository.LeasedOutboxMessage, error)
	MarkDelivered(context.Context, string, string, time.Time) error
	FailDelivery(context.Context, string, string, time.Time, time.Time, int, string) (domain.OutboxStatus, error)
}

// DeliveryAdapter performs one physical delivery attempt. Implementations must
// propagate Message.IdempotencyKey to the downstream consumer or provide an
// equivalent durable deduplication mechanism.
type DeliveryAdapter interface {
	Deliver(context.Context, domain.OutboxMessage) error
}

type DispatchResult struct {
	Leased     int
	Delivered  int
	Retried    int
	DeadLetter int
	Failed     int
}

type Dispatcher struct {
	store         Store
	owner         string
	leaseDuration time.Duration
	maxAttempts   int
	baseBackoff   time.Duration
	adapters      map[string]DeliveryAdapter
	now           func() time.Time
}

func NewDispatcher(store Store, owner string, leaseDuration time.Duration, maxAttempts int, baseBackoff time.Duration) (*Dispatcher, error) {
	owner = strings.TrimSpace(owner)
	if store == nil || owner == "" || leaseDuration <= 0 || maxAttempts < 1 || baseBackoff <= 0 {
		return nil, fmt.Errorf("store, owner, positive lease duration, max attempts and backoff are required")
	}
	return &Dispatcher{
		store: store, owner: owner, leaseDuration: leaseDuration,
		maxAttempts: maxAttempts, baseBackoff: baseBackoff,
		adapters: make(map[string]DeliveryAdapter),
		now:      func() time.Time { return time.Now().UTC() },
	}, nil
}

func (d *Dispatcher) Register(destination string, adapter DeliveryAdapter) error {
	destination = strings.TrimSpace(destination)
	if destination == "" || adapter == nil {
		return fmt.Errorf("destination and adapter are required")
	}
	if _, exists := d.adapters[destination]; exists {
		return fmt.Errorf("destination %q is already registered", destination)
	}
	d.adapters[destination] = adapter
	return nil
}

func (d *Dispatcher) DispatchOnce(ctx context.Context, limit int) (DispatchResult, error) {
	if d == nil || d.store == nil {
		return DispatchResult{}, fmt.Errorf("dispatcher is not configured")
	}
	now := d.now()
	leased, err := d.store.Lease(ctx, d.owner, limit, now, d.leaseDuration)
	if err != nil {
		return DispatchResult{}, err
	}
	result := DispatchResult{Leased: len(leased)}
	var joined error
	for _, item := range leased {
		adapter := d.adapters[item.Message.Destination]
		if adapter == nil {
			status, failErr := d.store.FailDelivery(
				ctx, item.Message.OutboxID, d.owner, now, now,
				item.Message.Attempts, ErrDestinationNotConfigured.Error(),
			)
			if failErr != nil {
				result.Failed++
				joined = errors.Join(joined, failErr)
				continue
			}
			if status == domain.OutboxDeadLetter {
				result.DeadLetter++
			}
			joined = errors.Join(joined, fmt.Errorf("%w: %s", ErrDestinationNotConfigured, item.Message.Destination))
			continue
		}

		if deliverErr := adapter.Deliver(ctx, item.Message); deliverErr != nil {
			next := now.Add(d.backoff(item.Message.Attempts))
			status, failErr := d.store.FailDelivery(
				ctx, item.Message.OutboxID, d.owner, now, next,
				d.maxAttempts, deliverErr.Error(),
			)
			if failErr != nil {
				result.Failed++
				joined = errors.Join(joined, failErr)
				continue
			}
			if status == domain.OutboxDeadLetter {
				result.DeadLetter++
			} else {
				result.Retried++
			}
			continue
		}

		if err := d.store.MarkDelivered(ctx, item.Message.OutboxID, d.owner, d.now()); err != nil {
			result.Failed++
			joined = errors.Join(joined, err)
			continue
		}
		result.Delivered++
	}
	return result, joined
}

func (d *Dispatcher) backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	backoff := d.baseBackoff
	for i := 1; i < attempt && backoff < 5*time.Minute; i++ {
		backoff *= 2
	}
	if backoff > 5*time.Minute {
		return 5 * time.Minute
	}
	return backoff
}
