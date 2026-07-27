package securityeval

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/approval"
	"github.com/tommyxie2026-tech/aicloud/internal/audit"
	"github.com/tommyxie2026-tech/aicloud/internal/credentials"
	"github.com/tommyxie2026-tech/aicloud/internal/policy"
	"github.com/tommyxie2026-tech/aicloud/internal/sandbox"
	"github.com/tommyxie2026-tech/aicloud/internal/toolgateway"
)

func TestPolicyDenyOccursBeforeCredentialIssuance(t *testing.T) {
	broker := newCountingBroker()
	auditStore := audit.NewMemoryStore()
	service := newService(policy.DenyByDefault{}, approval.NewMemoryStore(), broker, auditStore)
	_, err := service.Execute(context.Background(), validRequest("task-1"))
	if !errors.Is(err, toolgateway.ErrDenied) {
		t.Fatalf("expected denial, got %v", err)
	}
	if broker.issues != 0 {
		t.Fatalf("credentials issued before policy allow: %d", broker.issues)
	}
	events, _ := auditStore.ListByTask(context.Background(), "task-1")
	if len(events) != 1 || events[0].Status != "DENIED" {
		t.Fatalf("missing denial audit: %#v", events)
	}
}

func TestApprovalCannotBeReusedAcrossTasks(t *testing.T) {
	broker := newCountingBroker()
	approvals := approval.NewMemoryStore()
	_ = approvals.Put(context.Background(), approval.Record{
		ID: "approval-1", TaskID: "task-1", ToolID: "restricted-shell", Action: "execute",
		ApprovedBy: "reviewer", ApprovedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	service := newService(allowPolicy(true), approvals, broker, audit.NewMemoryStore())
	request := validRequest("task-2")
	request.ApprovalID = "approval-1"
	_, err := service.Execute(context.Background(), request)
	if !errors.Is(err, toolgateway.ErrApprovalRequired) {
		t.Fatalf("expected task-scoped approval rejection, got %v", err)
	}
	if broker.issues != 0 {
		t.Fatalf("credentials issued for mismatched approval: %d", broker.issues)
	}
}

func TestApprovedExecutionUsesHardenedSandboxAndRevokesCredential(t *testing.T) {
	broker := newCountingBroker()
	approvals := approval.NewMemoryStore()
	_ = approvals.Put(context.Background(), approval.Record{
		ID: "approval-1", TaskID: "task-1", ToolID: "restricted-shell", Action: "execute",
		ApprovedBy: "reviewer", ApprovedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	service := newService(allowPolicy(true), approvals, broker, audit.NewMemoryStore())
	request := validRequest("task-1")
	request.ApprovalID = "approval-1"
	result, err := service.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if broker.issues != 1 || broker.revokes != 1 {
		t.Fatalf("lease lifecycle issues=%d revokes=%d", broker.issues, broker.revokes)
	}
	if result.SandboxResult.Plan == nil {
		t.Fatal("expected sandbox plan")
	}
	jobSpec := result.SandboxResult.Plan.Job["spec"].(map[string]any)
	template := jobSpec["template"].(map[string]any)
	podSpec := template["spec"].(map[string]any)
	if podSpec["automountServiceAccountToken"] != false {
		t.Fatal("service account token must not be mounted")
	}
	containers := podSpec["containers"].([]any)
	container := containers[0].(map[string]any)
	securityContext := container["securityContext"].(map[string]any)
	if securityContext["privileged"] != false || securityContext["allowPrivilegeEscalation"] != false {
		t.Fatalf("sandbox privilege controls missing: %#v", securityContext)
	}
	networkSpec := result.SandboxResult.Plan.NetworkPolicy["spec"].(map[string]any)
	if len(networkSpec["egress"].([]any)) != 0 {
		t.Fatal("default sandbox must deny egress")
	}
}

func TestSandboxRejectsSecretEnvironmentAndWorkspaceEscape(t *testing.T) {
	service := newService(allowPolicy(false), approval.NewMemoryStore(), newCountingBroker(), audit.NewMemoryStore())
	secretRequest := validRequest("task-secret")
	secretRequest.Environment = map[string]string{"API_TOKEN": "do-not-inject"}
	if _, err := service.Execute(context.Background(), secretRequest); !errors.Is(err, sandbox.ErrInvalidRequest) {
		t.Fatalf("expected secret environment rejection, got %v", err)
	}

	escapeRequest := validRequest("task-escape")
	escapeRequest.WorkspacePath = "/tmp/../../etc"
	if _, err := service.Execute(context.Background(), escapeRequest); !errors.Is(err, sandbox.ErrInvalidRequest) {
		t.Fatalf("expected workspace escape rejection, got %v", err)
	}
}

func TestCredentialLeaseCannotCrossTasks(t *testing.T) {
	broker := credentials.NewMemoryBroker()
	lease, err := broker.Issue(context.Background(), credentials.Request{TaskID: "task-1", ToolID: "tool-1", Permission: "repo:read", TTL: time.Minute})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}
	if _, err := broker.Validate(context.Background(), lease.ID, "task-2", "tool-1"); !errors.Is(err, credentials.ErrInvalidLease) {
		t.Fatalf("expected cross-task lease rejection, got %v", err)
	}
}

func newService(engine policy.Engine, approvals approval.Store, broker credentials.Broker, auditStore audit.Store) *toolgateway.Service {
	registry := toolgateway.NewMemoryRegistry(toolgateway.Definition{
		ID: "restricted-shell", Version: "v1", Image: "alpine:3.20", Command: []string{"/bin/echo"},
		RiskLevel: "high", Permission: "workspace:write", CredentialTTL: time.Minute,
		CPU: "250m", Memory: "256Mi", Timeout: time.Minute, NetworkMode: sandbox.NetworkDenyAll,
		WorkspaceWrite: true,
	})
	return toolgateway.NewService(registry, engine, approvals, broker, sandbox.NewPlanningExecutor(), auditStore)
}

func allowPolicy(requiresApproval bool) policy.StaticEngine {
	return policy.StaticEngine{Version: "test-v1", Rules: []policy.Rule{{
		Name: "allow-shell", Subject: "agent-1", Action: "execute", Resource: "restricted-shell",
		Allowed: true, RequireApproval: requiresApproval, Reason: "test rule",
	}}}
}

func validRequest(taskID string) toolgateway.Request {
	return toolgateway.Request{
		TaskID: taskID, TraceID: "trace-1", AgentID: "agent-1", ToolID: "restricted-shell", Action: "execute",
		Arguments: []string{"hello"}, WorkspacePath: "/workspace/repo",
	}
}

type countingBroker struct {
	inner   *credentials.MemoryBroker
	issues  int
	revokes int
}

func newCountingBroker() *countingBroker {
	return &countingBroker{inner: credentials.NewMemoryBroker()}
}

func (b *countingBroker) Issue(ctx context.Context, request credentials.Request) (credentials.Lease, error) {
	b.issues++
	return b.inner.Issue(ctx, request)
}

func (b *countingBroker) Validate(ctx context.Context, leaseID, taskID, toolID string) (credentials.Lease, error) {
	return b.inner.Validate(ctx, leaseID, taskID, toolID)
}

func (b *countingBroker) Revoke(ctx context.Context, leaseID string) error {
	b.revokes++
	return b.inner.Revoke(ctx, leaseID)
}
