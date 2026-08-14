package workflow

import (
	"fmt"

	"go.temporal.io/sdk/activity"
	temporalworkflow "go.temporal.io/sdk/workflow"
)

type LifecycleRegistry interface {
	RegisterWorkflowWithOptions(interface{}, temporalworkflow.RegisterOptions)
	RegisterActivityWithOptions(interface{}, activity.RegisterOptions)
}

func RegisterLifecycle(registry LifecycleRegistry, activities LifecycleActivities) (err error) {
	if registry == nil {
		return fmt.Errorf("lifecycle registry is required")
	}
	if activities == nil {
		return fmt.Errorf("lifecycle activities are required")
	}

	// Temporal registration APIs panic on invalid or duplicate registration.
	// Convert that startup-time programming/configuration error into an explicit
	// fail-fast error for the worker entrypoint.
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("register Task lifecycle: %v", recovered)
		}
	}()

	registry.RegisterWorkflowWithOptions(TaskLifecycleWorkflow, temporalworkflow.RegisterOptions{
		Name: TaskExecutionWorkflowType,
	})
	registry.RegisterActivityWithOptions(activities.LoadTask, activity.RegisterOptions{Name: ActivityLoadTask})
	registry.RegisterActivityWithOptions(activities.TransitionTask, activity.RegisterOptions{Name: ActivityTransition})
	registry.RegisterActivityWithOptions(activities.PlanStub, activity.RegisterOptions{Name: ActivityPlanStub})
	registry.RegisterActivityWithOptions(activities.RouteStub, activity.RegisterOptions{Name: ActivityRouteStub})
	registry.RegisterActivityWithOptions(activities.ExecuteStub, activity.RegisterOptions{Name: ActivityExecuteStub})
	registry.RegisterActivityWithOptions(activities.ValidateStub, activity.RegisterOptions{Name: ActivityValidateStub})
	return nil
}
