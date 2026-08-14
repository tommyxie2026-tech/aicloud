package workflow

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"

	historypb "go.temporal.io/api/history/v1"
	temporalworker "go.temporal.io/sdk/worker"
	temporalworkflow "go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/proto"
)

func TestTaskLifecycleReplayCompatibility(t *testing.T) {
	encoded, err := os.ReadFile("testdata/s3c-terminal-history.pb.b64")
	if err != nil {
		t.Fatalf("read replay fixture: %v", err)
	}
	body, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil {
		t.Fatalf("decode replay fixture: %v", err)
	}
	history := &historypb.History{}
	if err := proto.Unmarshal(body, history); err != nil {
		t.Fatalf("unmarshal replay fixture: %v", err)
	}
	if len(history.Events) != 11 {
		t.Fatalf("replay fixture events=%d want=11", len(history.Events))
	}

	replayer := temporalworker.NewWorkflowReplayer()
	replayer.RegisterWorkflowWithOptions(TaskLifecycleWorkflow, temporalworkflow.RegisterOptions{
		Name: TaskExecutionWorkflowType,
	})
	if err := replayer.ReplayWorkflowHistory(nil, history); err != nil {
		t.Fatalf("Task lifecycle history is no longer replay compatible: %v", err)
	}
}
