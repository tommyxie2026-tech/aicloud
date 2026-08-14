package config

import "testing"

func TestTemporalConfigDisabledIsValid(t *testing.T) {
	if err := (TemporalConfig{}).Validate(); err != nil {
		t.Fatalf("disabled Temporal config rejected: %v", err)
	}
}

func TestTemporalConfigEnabledRequiresExecutionBoundary(t *testing.T) {
	valid := TemporalConfig{
		Enabled:                  true,
		Address:                  "localhost:7233",
		Namespace:                "default",
		TaskQueue:                "aicloud-task-v1",
		WorkerStopTimeoutSeconds: 30,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid Temporal config rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*TemporalConfig)
	}{
		{name: "address", mutate: func(c *TemporalConfig) { c.Address = "" }},
		{name: "namespace", mutate: func(c *TemporalConfig) { c.Namespace = "" }},
		{name: "task queue", mutate: func(c *TemporalConfig) { c.TaskQueue = "" }},
		{name: "stop timeout", mutate: func(c *TemporalConfig) { c.WorkerStopTimeoutSeconds = 0 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := valid
			tc.mutate(&config)
			if err := config.Validate(); err == nil {
				t.Fatal("invalid enabled Temporal config was accepted")
			}
		})
	}
}

func TestLoadTemporalDefaultsDisabled(t *testing.T) {
	t.Setenv("AICLOUD_TEMPORAL_ENABLED", "")
	t.Setenv("AICLOUD_TEMPORAL_ADDRESS", "")
	t.Setenv("AICLOUD_TEMPORAL_NAMESPACE", "")
	t.Setenv("AICLOUD_TEMPORAL_TASK_QUEUE", "")
	t.Setenv("AICLOUD_TEMPORAL_WORKER_STOP_TIMEOUT_SECONDS", "")

	config := Load().Temporal
	if config.Enabled {
		t.Fatal("Temporal must be disabled by default")
	}
	if config.Address != "localhost:7233" || config.Namespace != "default" || config.TaskQueue != "aicloud-task-v1" || config.WorkerStopTimeoutSeconds != 30 {
		t.Fatalf("unexpected Temporal defaults: %+v", config)
	}
}
