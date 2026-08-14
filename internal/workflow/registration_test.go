package workflow

import (
	"testing"

	"go.temporal.io/sdk/activity"
	temporalworkflow "go.temporal.io/sdk/workflow"
)

type recordingLifecycleRegistry struct {
	workflowNames []string
	activityNames []string
}

func (r *recordingLifecycleRegistry) RegisterWorkflowWithOptions(_ interface{}, options temporalworkflow.RegisterOptions) {
	r.workflowNames = append(r.workflowNames, options.Name)
}

func (r *recordingLifecycleRegistry) RegisterActivityWithOptions(_ interface{}, options activity.RegisterOptions) {
	r.activityNames = append(r.activityNames, options.Name)
}

func TestRegisterLifecycleUsesStableExternalNames(t *testing.T) {
	registry := &recordingLifecycleRegistry{}
	if err := RegisterLifecycle(registry, FailClosedLifecycleActivities{}); err != nil {
		t.Fatal(err)
	}
	if len(registry.workflowNames) != 1 || registry.workflowNames[0] != TaskExecutionWorkflowType {
		t.Fatalf("workflow names=%v", registry.workflowNames)
	}
	expected := []string{
		ActivityLoadTask,
		ActivityTransition,
		ActivityPlanStub,
		ActivityRouteStub,
		ActivityExecuteStub,
		ActivityValidateStub,
	}
	if len(registry.activityNames) != len(expected) {
		t.Fatalf("activity names=%v", registry.activityNames)
	}
	for index := range expected {
		if registry.activityNames[index] != expected[index] {
			t.Fatalf("activity[%d]=%q want=%q", index, registry.activityNames[index], expected[index])
		}
	}
}

func TestRegisterLifecycleFailsClosedWithoutDependencies(t *testing.T) {
	if err := RegisterLifecycle(nil, FailClosedLifecycleActivities{}); err == nil {
		t.Fatal("nil registry accepted")
	}
	if err := RegisterLifecycle(&recordingLifecycleRegistry{}, nil); err == nil {
		t.Fatal("nil activities accepted")
	}
}
