package workflow

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	temporalworker "go.temporal.io/sdk/worker"
	"google.golang.org/protobuf/encoding/protojson"
)

// TestGenerateS3CReplayFixture is temporary. It intentionally fails after
// printing a gzip/base64 encoded real Temporal history so the stable fixture
// can be checked in and this generator removed before merge.
func TestGenerateS3CReplayFixture(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	server, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{
		CachedDownload: testsuite.CachedDownload{Version: "default"},
		LogLevel:       "error",
	})
	if err != nil {
		t.Fatalf("start Temporal dev server: %v", err)
	}
	defer func() {
		if stopErr := server.Stop(); stopErr != nil {
			t.Logf("stop Temporal dev server: %v", stopErr)
		}
	}()

	activities := NewMemoryLifecycleActivities(newLifecycleTask("CANCELLED"))
	queue := "s3c-replay-fixture"
	w := temporalworker.New(server.Client(), queue, temporalworker.Options{DisableRegistrationAliasing: true})
	if err := RegisterLifecycle(w, activities); err != nil {
		t.Fatalf("register lifecycle: %v", err)
	}
	if err := w.Start(); err != nil {
		t.Fatalf("start worker: %v", err)
	}
	defer w.Stop()

	run, err := server.Client().ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        "s3c-replay-terminal-task-a",
		TaskQueue: queue,
	}, TaskExecutionWorkflowType, lifecycleInput())
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	var result TaskWorkflowResult
	if err := run.Get(ctx, &result); err != nil {
		t.Fatalf("get workflow result: %v", err)
	}
	if !result.AlreadyTerminal {
		t.Fatalf("fixture workflow did not take terminal path: %+v", result)
	}

	iterator := server.Client().GetWorkflowHistory(
		ctx,
		run.GetID(),
		run.GetRunID(),
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)
	history := &historypb.History{}
	for iterator.HasNext() {
		event, nextErr := iterator.Next()
		if nextErr != nil {
			t.Fatalf("read workflow history: %v", nextErr)
		}
		history.Events = append(history.Events, event)
	}

	body, err := protojson.MarshalOptions{Indent: "  "}.Marshal(history)
	if err != nil {
		t.Fatalf("marshal workflow history: %v", err)
	}
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	if _, err := zw.Write(body); err != nil {
		t.Fatalf("compress workflow history: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close workflow history compressor: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(compressed.Bytes())
	t.Fatalf("S3C_REPLAY_FIXTURE_BEGIN\n%s\nS3C_REPLAY_FIXTURE_END\njson_bytes=%d events=%d", encoded, len(body), len(history.Events))
}

var _ = fmt.Sprintf
