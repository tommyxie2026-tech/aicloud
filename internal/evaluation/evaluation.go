package evaluation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

type Config struct {
	ModelID            string            `json:"modelId"`
	ModelVersion       string            `json:"modelVersion"`
	Provider           string            `json:"provider"`
	EndpointID         string            `json:"endpointId,omitempty"`
	PromptVersion      string            `json:"promptVersion"`
	WorkflowVersion    string            `json:"workflowVersion"`
	ToolVersions       map[string]string `json:"toolVersions,omitempty"`
	PermissionVersion string            `json:"permissionVersion,omitempty"`
	TokenBudget        int               `json:"tokenBudget"`
	TimeBudgetMS       int64             `json:"timeBudgetMs"`
	RetryPolicyVersion string            `json:"retryPolicyVersion"`
	CompactionVersion  string            `json:"compactionVersion,omitempty"`
	SandboxProfile     string            `json:"sandboxProfile"`
	DatasetID          string            `json:"datasetId"`
	DatasetVersion     string            `json:"datasetVersion"`
	EvaluatorID        string            `json:"evaluatorId"`
	EvaluatorVersion   string            `json:"evaluatorVersion"`
	Parameters         map[string]string `json:"parameters,omitempty"`
}

type Metrics struct {
	QualityScore          float64 `json:"qualityScore"`
	SafetyScore           float64 `json:"safetyScore"`
	ReliabilityScore      float64 `json:"reliabilityScore"`
	LatencyP95MS          int64   `json:"latencyP95Ms"`
	CostPerSuccessfulTask float64 `json:"costPerSuccessfulTask"`
	HumanInterventionRate float64 `json:"humanInterventionRate"`
	TaskSuccessRate       float64 `json:"taskSuccessRate"`
}

type Thresholds struct {
	MinimumQuality          float64 `json:"minimumQuality"`
	MinimumSafety           float64 `json:"minimumSafety"`
	MinimumReliability      float64 `json:"minimumReliability"`
	MaximumLatencyP95MS     int64   `json:"maximumLatencyP95Ms,omitempty"`
	MaximumCostPerSuccess   float64 `json:"maximumCostPerSuccess,omitempty"`
	MaximumHumanIntervention float64 `json:"maximumHumanIntervention,omitempty"`
	MinimumTaskSuccessRate  float64 `json:"minimumTaskSuccessRate"`
}

type GateResult struct {
	Passed  bool     `json:"passed"`
	Reasons []string `json:"reasons,omitempty"`
}

type Run struct {
	ID             string     `json:"id"`
	TaskID         string     `json:"taskId"`
	TraceID        string     `json:"traceId"`
	Config         Config     `json:"config"`
	ConfigDigest   string     `json:"configDigest"`
	RawOutputDigest string    `json:"rawOutputDigest,omitempty"`
	Metrics        Metrics    `json:"metrics"`
	Thresholds     Thresholds `json:"thresholds"`
	Gate           GateResult `json:"gate"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type Store interface {
	Append(context.Context, Run) error
	ListByTask(context.Context, string) ([]Run, error)
	Get(context.Context, string) (Run, error)
}

type MemoryStore struct {
	mu   sync.RWMutex
	runs map[string]Run
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{runs: make(map[string]Run)} }

func (s *MemoryStore) Append(_ context.Context, run Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.runs[run.ID]; exists {
		return fmt.Errorf("evaluation run already exists")
	}
	s.runs[run.ID] = cloneRun(run)
	return nil
}

func (s *MemoryStore) ListByTask(_ context.Context, taskID string) ([]Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Run, 0)
	for _, run := range s.runs {
		if run.TaskID == taskID {
			items = append(items, cloneRun(run))
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[id]
	if !ok {
		return Run{}, fmt.Errorf("evaluation run not found")
	}
	return cloneRun(run), nil
}

func NewRun(taskID, traceID string, config Config, metrics Metrics, thresholds Thresholds, rawOutput []byte, now time.Time) (Run, error) {
	if taskID == "" || traceID == "" {
		return Run{}, fmt.Errorf("task ID and trace ID are required")
	}
	if err := ValidateConfig(config); err != nil {
		return Run{}, err
	}
	configDigest, err := DigestConfig(config)
	if err != nil {
		return Run{}, err
	}
	rawDigest := ""
	if len(rawOutput) > 0 {
		sum := sha256.Sum256(rawOutput)
		rawDigest = "sha256:" + hex.EncodeToString(sum[:])
	}
	return Run{
		ID:              "eval-" + stringsafe(configDigest),
		TaskID:          taskID,
		TraceID:         traceID,
		Config:          cloneConfig(config),
		ConfigDigest:    configDigest,
		RawOutputDigest: rawDigest,
		Metrics:         metrics,
		Thresholds:      thresholds,
		Gate:            EvaluateGate(metrics, thresholds),
		CreatedAt:       now.UTC(),
	}, nil
}

func ValidateConfig(config Config) error {
	required := map[string]string{
		"model ID":             config.ModelID,
		"model version":        config.ModelVersion,
		"provider":             config.Provider,
		"prompt version":       config.PromptVersion,
		"workflow version":     config.WorkflowVersion,
		"retry policy version": config.RetryPolicyVersion,
		"sandbox profile":      config.SandboxProfile,
		"dataset ID":           config.DatasetID,
		"dataset version":      config.DatasetVersion,
		"evaluator ID":         config.EvaluatorID,
		"evaluator version":    config.EvaluatorVersion,
	}
	for name, value := range required {
		if value == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if config.TokenBudget <= 0 || config.TimeBudgetMS <= 0 {
		return fmt.Errorf("positive token and time budgets are required")
	}
	return nil
}

func DigestConfig(config Config) (string, error) {
	canonical := canonicalConfig(config)
	body, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("encode evaluation config: %w", err)
	}
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func EvaluateGate(metrics Metrics, thresholds Thresholds) GateResult {
	reasons := make([]string, 0)
	if metrics.QualityScore < thresholds.MinimumQuality {
		reasons = append(reasons, "quality score below threshold")
	}
	if metrics.SafetyScore < thresholds.MinimumSafety {
		reasons = append(reasons, "safety score below threshold")
	}
	if metrics.ReliabilityScore < thresholds.MinimumReliability {
		reasons = append(reasons, "reliability score below threshold")
	}
	if thresholds.MaximumLatencyP95MS > 0 && metrics.LatencyP95MS > thresholds.MaximumLatencyP95MS {
		reasons = append(reasons, "latency exceeds threshold")
	}
	if thresholds.MaximumCostPerSuccess > 0 && metrics.CostPerSuccessfulTask > thresholds.MaximumCostPerSuccess {
		reasons = append(reasons, "cost per successful task exceeds threshold")
	}
	if thresholds.MaximumHumanIntervention > 0 && metrics.HumanInterventionRate > thresholds.MaximumHumanIntervention {
		reasons = append(reasons, "human intervention rate exceeds threshold")
	}
	if metrics.TaskSuccessRate < thresholds.MinimumTaskSuccessRate {
		reasons = append(reasons, "task success rate below threshold")
	}
	return GateResult{Passed: len(reasons) == 0, Reasons: reasons}
}

type canonicalKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type canonicalEvaluationConfig struct {
	ModelID            string        `json:"modelId"`
	ModelVersion       string        `json:"modelVersion"`
	Provider           string        `json:"provider"`
	EndpointID         string        `json:"endpointId,omitempty"`
	PromptVersion      string        `json:"promptVersion"`
	WorkflowVersion    string        `json:"workflowVersion"`
	ToolVersions       []canonicalKV `json:"toolVersions,omitempty"`
	PermissionVersion string        `json:"permissionVersion,omitempty"`
	TokenBudget        int           `json:"tokenBudget"`
	TimeBudgetMS       int64         `json:"timeBudgetMs"`
	RetryPolicyVersion string        `json:"retryPolicyVersion"`
	CompactionVersion  string        `json:"compactionVersion,omitempty"`
	SandboxProfile     string        `json:"sandboxProfile"`
	DatasetID          string        `json:"datasetId"`
	DatasetVersion     string        `json:"datasetVersion"`
	EvaluatorID        string        `json:"evaluatorId"`
	EvaluatorVersion   string        `json:"evaluatorVersion"`
	Parameters         []canonicalKV `json:"parameters,omitempty"`
}

func canonicalConfig(config Config) canonicalEvaluationConfig {
	return canonicalEvaluationConfig{
		ModelID:            config.ModelID,
		ModelVersion:       config.ModelVersion,
		Provider:           config.Provider,
		EndpointID:         config.EndpointID,
		PromptVersion:      config.PromptVersion,
		WorkflowVersion:    config.WorkflowVersion,
		ToolVersions:       sortedKV(config.ToolVersions),
		PermissionVersion: config.PermissionVersion,
		TokenBudget:        config.TokenBudget,
		TimeBudgetMS:       config.TimeBudgetMS,
		RetryPolicyVersion: config.RetryPolicyVersion,
		CompactionVersion:  config.CompactionVersion,
		SandboxProfile:     config.SandboxProfile,
		DatasetID:          config.DatasetID,
		DatasetVersion:     config.DatasetVersion,
		EvaluatorID:        config.EvaluatorID,
		EvaluatorVersion:   config.EvaluatorVersion,
		Parameters:         sortedKV(config.Parameters),
	}
}

func sortedKV(values map[string]string) []canonicalKV {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]canonicalKV, 0, len(keys))
	for _, key := range keys {
		items = append(items, canonicalKV{Key: key, Value: values[key]})
	}
	return items
}

func cloneConfig(config Config) Config {
	config.ToolVersions = cloneMap(config.ToolVersions)
	config.Parameters = cloneMap(config.Parameters)
	return config
}

func cloneRun(run Run) Run {
	run.Config = cloneConfig(run.Config)
	run.Gate.Reasons = append([]string(nil), run.Gate.Reasons...)
	return run
}

func cloneMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func stringsafe(value string) string {
	if len(value) > 20 {
		value = value[len(value)-20:]
	}
	return value
}
