package sandbox

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var ErrInvalidRequest = errors.New("invalid sandbox request")

type NetworkMode string

const (
	NetworkDenyAll   NetworkMode = "deny-all"
	NetworkAllowList NetworkMode = "allow-list"
)

type Limits struct {
	CPU            string        `json:"cpu"`
	Memory         string        `json:"memory"`
	Timeout        time.Duration `json:"timeout"`
	NetworkMode    NetworkMode   `json:"networkMode"`
	AllowedHosts   []string      `json:"allowedHosts,omitempty"`
	ReadOnlyRootFS bool          `json:"readOnlyRootFs"`
	WorkspacePath  string        `json:"workspacePath"`
	WorkspaceWrite bool          `json:"workspaceWrite"`
}

type Request struct {
	TaskID            string            `json:"taskId"`
	TraceID           string            `json:"traceId"`
	ToolID            string            `json:"toolId"`
	Image             string            `json:"image"`
	Command           []string          `json:"command"`
	Environment       map[string]string `json:"environment,omitempty"`
	CredentialLeaseID string            `json:"credentialLeaseId,omitempty"`
	Limits            Limits            `json:"limits"`
}

type Plan struct {
	Namespace      string         `json:"namespace"`
	ServiceAccount string         `json:"serviceAccount"`
	Job            map[string]any `json:"job"`
	NetworkPolicy  map[string]any `json:"networkPolicy"`
}

type Result struct {
	Status      string            `json:"status"`
	ExitCode    int               `json:"exitCode,omitempty"`
	Stdout      string            `json:"stdout,omitempty"`
	Stderr      string            `json:"stderr,omitempty"`
	Artifacts   map[string]string `json:"artifacts,omitempty"`
	Plan        *Plan             `json:"plan,omitempty"`
	StartedAt   time.Time         `json:"startedAt"`
	CompletedAt time.Time         `json:"completedAt"`
}

type Executor interface {
	Execute(context.Context, Request) (Result, error)
}

type Planner struct{}

func (Planner) Plan(request Request) (Plan, error) {
	if err := validate(request); err != nil {
		return Plan{}, err
	}
	name := dnsName(request.TaskID)
	namespace := "aicloud-task-" + name
	serviceAccount := "sandbox-" + name
	deadline := int64(request.Limits.Timeout.Seconds())
	job := map[string]any{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]any{
			"name":      "tool-" + name,
			"namespace": namespace,
			"labels":    map[string]string{"aicloud.dev/task-id": request.TaskID, "aicloud.dev/tool-id": request.ToolID},
		},
		"spec": map[string]any{
			"backoffLimit":            0,
			"activeDeadlineSeconds":   deadline,
			"ttlSecondsAfterFinished": 300,
			"template": map[string]any{
				"metadata": map[string]any{"labels": map[string]string{"aicloud.dev/task-id": request.TaskID}},
				"spec": map[string]any{
					"restartPolicy":                "Never",
					"serviceAccountName":           serviceAccount,
					"automountServiceAccountToken": false,
					"securityContext": map[string]any{
						"runAsNonRoot":   true,
						"seccompProfile": map[string]string{"type": "RuntimeDefault"},
					},
					"containers": []any{map[string]any{
						"name":    "tool",
						"image":   request.Image,
						"command": request.Command,
						"env":     safeEnvironment(request.Environment),
						"securityContext": map[string]any{
							"allowPrivilegeEscalation": false,
							"privileged":               false,
							"readOnlyRootFilesystem":   request.Limits.ReadOnlyRootFS,
							"capabilities":             map[string]any{"drop": []string{"ALL"}},
						},
						"resources": map[string]any{
							"requests": map[string]string{"cpu": request.Limits.CPU, "memory": request.Limits.Memory},
							"limits":   map[string]string{"cpu": request.Limits.CPU, "memory": request.Limits.Memory},
						},
					}},
				},
			},
		},
	}
	networkPolicy := map[string]any{
		"apiVersion": "networking.k8s.io/v1",
		"kind":       "NetworkPolicy",
		"metadata":   map[string]any{"name": "sandbox-network", "namespace": namespace},
		"spec": map[string]any{
			"podSelector": map[string]any{},
			"policyTypes": []string{"Ingress", "Egress"},
			"ingress":     []any{},
			"egress":      networkEgress(request.Limits),
		},
	}
	return Plan{Namespace: namespace, ServiceAccount: serviceAccount, Job: job, NetworkPolicy: networkPolicy}, nil
}

type PlanningExecutor struct {
	Planner Planner
	now     func() time.Time
}

func NewPlanningExecutor() *PlanningExecutor {
	return &PlanningExecutor{Planner: Planner{}, now: time.Now}
}

func (e *PlanningExecutor) Execute(_ context.Context, request Request) (Result, error) {
	plan, err := e.Planner.Plan(request)
	if err != nil {
		return Result{}, err
	}
	now := e.now().UTC()
	return Result{Status: "PLANNED", Plan: &plan, StartedAt: now, CompletedAt: now}, nil
}

func validate(request Request) error {
	if request.TaskID == "" || request.TraceID == "" || request.ToolID == "" || request.Image == "" || len(request.Command) == 0 {
		return ErrInvalidRequest
	}
	if request.Limits.Timeout <= 0 || request.Limits.Timeout > time.Hour {
		return fmt.Errorf("%w: timeout must be between 1 second and 1 hour", ErrInvalidRequest)
	}
	if request.Limits.CPU == "" || request.Limits.Memory == "" {
		return fmt.Errorf("%w: CPU and memory limits are required", ErrInvalidRequest)
	}
	if request.Limits.NetworkMode == "" {
		request.Limits.NetworkMode = NetworkDenyAll
	}
	if request.Limits.NetworkMode != NetworkDenyAll && request.Limits.NetworkMode != NetworkAllowList {
		return fmt.Errorf("%w: unsupported network mode", ErrInvalidRequest)
	}
	if request.Limits.NetworkMode == NetworkAllowList && len(request.Limits.AllowedHosts) == 0 {
		return fmt.Errorf("%w: allow-list network mode requires allowed hosts", ErrInvalidRequest)
	}
	if request.Limits.WorkspacePath != "" {
		clean := filepath.Clean(request.Limits.WorkspacePath)
		if !strings.HasPrefix(clean, "/workspace") || strings.Contains(clean, "..") {
			return fmt.Errorf("%w: workspace must remain under /workspace", ErrInvalidRequest)
		}
	}
	for key := range request.Environment {
		upper := strings.ToUpper(key)
		if strings.Contains(upper, "SECRET") || strings.Contains(upper, "TOKEN") || strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "KEY") {
			return fmt.Errorf("%w: raw secret-like environment variables are prohibited", ErrInvalidRequest)
		}
	}
	return nil
}

func safeEnvironment(values map[string]string) []map[string]string {
	items := make([]map[string]string, 0, len(values))
	for name, value := range values {
		items = append(items, map[string]string{"name": name, "value": value})
	}
	return items
}

func networkEgress(limits Limits) []any {
	if limits.NetworkMode != NetworkAllowList {
		return []any{}
	}
	// Hostnames are retained as policy evidence. A production controller must
	// resolve them through an approved egress proxy rather than converting them
	// directly into broad CIDR rules.
	return []any{map[string]any{
		"to":    []any{map[string]any{"namespaceSelector": map[string]any{"matchLabels": map[string]string{"aicloud.dev/egress-proxy": "true"}}}},
		"ports": []any{map[string]any{"protocol": "TCP", "port": 443}},
	}}
}

var invalidDNS = regexp.MustCompile(`[^a-z0-9-]+`)

func dnsName(value string) string {
	value = strings.ToLower(value)
	value = invalidDNS.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if len(value) > 40 {
		value = value[:40]
	}
	if value == "" {
		return "task"
	}
	return value
}
