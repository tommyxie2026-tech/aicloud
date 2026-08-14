package repository

import (
	"context"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func TestMemoryDeploymentLifecycleEventsRetainEvidence(t *testing.T) {
	store := NewMemoryDeploymentLifecycleEvents()
	announced := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
	effective := announced.Add(24 * time.Hour)
	quota := int64(100)
	event := domain.DeploymentLifecycleEvent{
		ID: "event-1", DeploymentID: "deployment-1",
		From: domain.DeploymentReady, To: domain.DeploymentDraining,
		AnnouncedAt: &announced, EffectiveAt: effective,
		EvidenceRef: "provider-notice-1", ReplacementIDs: []string{"deployment-2"},
		QuotaRemaining: &quota, RateLimitRef: "rate-limit-v2",
		RoutingEligible: false, MigrationState: "planned", CreatedAt: announced,
	}
	if _, err := store.Append(context.Background(), event); err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	items, err := store.ListByDeployment(context.Background(), "deployment-1")
	if err != nil {
		t.Fatalf("ListByDeployment returned error: %v", err)
	}
	if len(items) != 1 || items[0].EvidenceRef != event.EvidenceRef || items[0].ReplacementIDs[0] != "deployment-2" || items[0].QuotaRemaining == nil || *items[0].QuotaRemaining != quota {
		t.Fatalf("lifecycle evidence changed: %#v", items)
	}
}
