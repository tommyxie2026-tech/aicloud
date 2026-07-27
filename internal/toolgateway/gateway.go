package toolgateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/approval"
	"github.com/tommyxie2026-tech/aicloud/internal/audit"
	"github.com/tommyxie2026-tech/aicloud/internal/credentials"
	"github.com/tommyxie2026-tech/aicloud/internal/policy"
	"github.com/tommyxie2026-tech/aicloud/internal/sandbox"
)

var (
	ErrDenied           = errors.New("tool invocation denied")
	ErrApprovalRequired = errors.New("tool invocation requires approval")
	ErrToolNotFound     = errors.New("tool not found")
)

type Gateway interface {
	Invoke(context.Context, string, string) (string, error)
}

type DenyByDefault struct{}

func (DenyByDefault) Invoke(context.Context, string, string) (string, error) {
	return "", errors.New("tool execution is disabled in the skeleton runtime")
}

type Definition struct {
	ID             string              `json:"id"`
	Version        string              `json:"version"`
	Image          string              `json:"image"`
	Command        []string            `json:"command"`
	RiskLevel      string              `json:"riskLevel"`
	Permission     string              `json:"permission"`
	CredentialTTL  time.Duration       `json:"credentialTtl"`
	CPU            string              `json:"cpu"`
	Memory         string              `json:"memory"`
	Timeout        time.Duration       `json:"timeout"`
	NetworkMode    sandbox.NetworkMode `json:"networkMode"`
	AllowedHosts   []string            `json:"allowedHosts,omitempty"`
	WorkspaceWrite bool                `json:"workspaceWrite"`
}

type Registry interface {
	Get(context.Context, string) (Definition, error)
	Put(context.Context, Definition) error
	List(context.Context) ([]Definition, error)
}

type MemoryRegistry struct {
	mu    sync.RWMutex
	tools map[string]Definition
}

func NewMemoryRegistry(definitions ...Definition) *MemoryRegistry {
	registry := &MemoryRegistry{tools: make(map[string]Definition)}
	for _, definition := range definitions {
		registry.tools[definition.ID] = definition
	}
	return registry
}

func (r *MemoryRegistry) Get(_ context.Context, id string) (Definition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	definition, ok := r.tools[id]
	if !ok {
		return Definition{}, ErrToolNotFound
	}
	return definition, nil
}

func (r *MemoryRegistry) Put(_ context.Context, definition Definition) error {
	if definition.ID == "" || definition.Image == "" || len(definition.Command) == 0 || definition.Permission == "" {
		return fmt.Errorf("invalid tool definition")
	}
	r.mu.Lock()
	r.tools[definition.ID] = definition
	r.mu.Unlock()
	return nil
}

func (r *MemoryRegistry) List(_ context.Context) ([]Definition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]Definition, 0, len(r.tools))
	for _, definition := range r.tools {
		items = append(items, definition)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

type Request struct {
	TaskID       string            `json:"taskId"`
	TraceID      string            `json:"traceId"`
	AgentID      string            `json:"agentId"`
	ToolID       string            `json:"toolId"`
	Action       string            `json:"action"`
	Arguments    []string          `json:"arguments,omitempty"`
	Environment  map[string]string `json:"environment,omitempty"`
	ApprovalID   string            `json:"approvalId,omitempty"`
	WorkspacePath string           `json:"workspacePath,omitempty"`
}

type Result struct {
	ToolID        string          `json:"toolId"`
	Policy        policy.Decision `json:"policy"`
	ApprovalID    string          `json:"approvalId,omitempty"`
	CredentialID  string          `json:"credentialId,omitempty"`
	SandboxResult sandbox.Result  `json:"sandboxResult"`
}

type Service struct {
	registry    Registry
	policy      policy.Engine
	approvals   approval.Store
	credentials credentials.Broker
	sandbox     sandbox.Executor
	audit       audit.Store
	now         func() time.Time
}

func NewService(registry Registry, policyEngine policy.Engine, approvals approval.Store, broker credentials.Broker, executor sandbox.Executor, auditStore audit.Store) *Service {
	return &Service{registry: registry, policy: policyEngine, approvals: approvals, credentials: broker, sandbox: executor, audit: auditStore, now: time.Now}
}

func (s *Service) Execute(ctx context.Context, request Request) (Result, error) {
	if request.TaskID == "" || request.TraceID == "" || request.AgentID == "" || request.ToolID == "" || request.Action == "" {
		return Result{}, fmt.Errorf("invalid tool request")
	}
	if s.registry == nil || s.policy == nil || s.approvals == nil || s.credentials == nil || s.sandbox == nil {
		return Result{}, fmt.Errorf("tool gateway dependencies are incomplete")
	}
	definition, err := s.registry.Get(ctx, request.ToolID)
	if err != nil {
		return Result{}, err
	}
	decision, err := s.policy.Evaluate(ctx, request.AgentID, request.Action, request.ToolID)
	if err != nil {
		return Result{}, err
	}
	if !decision.Allowed {
		s.appendAudit(ctx, request, definition, decision, "DENIED", decision.Reason, "", "", nil)
		return Result{ToolID: request.ToolID, Policy: decision}, ErrDenied
	}
	if decision.RequiresApproval {
		if request.ApprovalID == "" {
			s.appendAudit(ctx, request, definition, decision, "WAITING_APPROVAL", "approval is required", "", "", nil)
			return Result{ToolID: request.ToolID, Policy: decision}, ErrApprovalRequired
		}
		if _, err := s.approvals.Validate(ctx, request.ApprovalID, request.TaskID, request.ToolID, request.Action); err != nil {
			s.appendAudit(ctx, request, definition, decision, "DENIED", "approval is invalid", "", "", nil)
			return Result{ToolID: request.ToolID, Policy: decision}, ErrApprovalRequired
		}
	}

	command := append(append([]string(nil), definition.Command...), request.Arguments...)
	sandboxRequest := sandbox.Request{
		TaskID:      request.TaskID,
		TraceID:     request.TraceID,
		ToolID:      request.ToolID,
		Image:       definition.Image,
		Command:     command,
		Environment: request.Environment,
		Limits: sandbox.Limits{
			CPU:            definition.CPU,
			Memory:         definition.Memory,
			Timeout:        definition.Timeout,
			NetworkMode:    definition.NetworkMode,
			AllowedHosts:   definition.AllowedHosts,
			ReadOnlyRootFS: true,
			WorkspacePath:  request.WorkspacePath,
			WorkspaceWrite: definition.WorkspaceWrite,
		},
	}
	if _, err := (sandbox.Planner{}).Plan(sandboxRequest); err != nil {
		s.appendAudit(ctx, request, definition, decision, "DENIED", err.Error(), request.ApprovalID, "", nil)
		return Result{}, err
	}

	lease, err := s.credentials.Issue(ctx, credentials.Request{TaskID: request.TaskID, ToolID: request.ToolID, Permission: definition.Permission, TTL: definition.CredentialTTL})
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = s.credentials.Revoke(context.Background(), lease.ID) }()
	sandboxRequest.CredentialLeaseID = lease.ID
	sandboxResult, err := s.sandbox.Execute(ctx, sandboxRequest)
	if err != nil {
		s.appendAudit(ctx, request, definition, decision, "FAILED", err.Error(), request.ApprovalID, lease.ID, nil)
		return Result{}, err
	}
	result := Result{ToolID: request.ToolID, Policy: decision, ApprovalID: request.ApprovalID, CredentialID: lease.ID, SandboxResult: sandboxResult}
	s.appendAudit(ctx, request, definition, decision, sandboxResult.Status, "", request.ApprovalID, lease.ID, result)
	return result, nil
}

func (s *Service) appendAudit(ctx context.Context, request Request, definition Definition, decision policy.Decision, status, reason, approvalID, leaseID string, result any) {
	if s.audit == nil {
		return
	}
	now := s.now().UTC()
	event := audit.Event{
		ID:              fmt.Sprintf("audit-%d", now.UnixNano()),
		TaskID:          request.TaskID,
		TraceID:         request.TraceID,
		AgentID:         request.AgentID,
		ToolID:          definition.ID,
		Action:          request.Action,
		PolicyAllowed:   decision.Allowed,
		PolicyVersion:   decision.PolicyVersion,
		MatchedRule:     decision.MatchedRule,
		ApprovalID:      approvalID,
		CredentialLease: leaseID,
		InputDigest:     digest(request),
		ResultDigest:    digest(result),
		Status:          status,
		Reason:          reason,
		CreatedAt:       now,
	}
	_ = s.audit.Append(ctx, event)
}

func digest(value any) string {
	if value == nil {
		return ""
	}
	body, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
