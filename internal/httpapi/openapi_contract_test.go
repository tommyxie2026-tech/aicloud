package httpapi

import (
	"bufio"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestOpenAPIOperations(t *testing.T) {
	body := readOpenAPI(t)
	actual := parseOperations(t, body)
	expected := []string{
		"GET /models", "POST /models", "GET /models/{model_id}", "PUT /models/{model_id}",
		"GET /models/{model_id}/admission", "POST /models/{model_id}/admission", "GET /tools",
		"GET /tasks", "POST /tasks", "GET /tasks/{task_id}", "POST /tasks/{task_id}/route",
		"GET /tasks/{task_id}/routes", "GET /tasks/{task_id}/costs", "GET /tasks/{task_id}/audit",
		"POST /tasks/{task_id}/model", "GET /tasks/{task_id}/trace", "GET /tasks/{task_id}/evaluations",
		"POST /tasks/{task_id}/evaluations", "POST /tasks/{task_id}/tools/{tool_id}",
	}
	sort.Strings(actual)
	sort.Strings(expected)
	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("OpenAPI operation drift\nactual=%v\nexpected=%v", actual, expected)
	}
}

func TestOpenAPITaskStatesAndCommandHeaders(t *testing.T) {
	body := readOpenAPI(t)
	for _, state := range []string{"CREATED", "PLANNING", "ROUTING", "EXECUTING", "WAITING_APPROVAL", "VALIDATING", "COMPLETED", "FAILED", "CANCELLED", "EXPIRED"} {
		if !strings.Contains(body, state) {
			t.Fatalf("missing Task state %s", state)
		}
	}
	for _, marker := range []string{"  /tasks:\n", "  /tasks/{task_id}/route:\n", "  /tasks/{task_id}/model:\n"} {
		section := pathSection(t, body, marker)
		if !strings.Contains(section, "#/components/parameters/IdempotencyKey") {
			t.Fatalf("%s is missing Idempotency-Key", strings.TrimSpace(marker))
		}
	}
}

func TestOpenAPICoreRequestsAreClosed(t *testing.T) {
	body := readOpenAPI(t)
	for _, name := range []string{"CreateTaskRequest:", "RouteRequest:", "ModelExecutionRequest:", "EvaluationRequest:", "ToolExecutionRequest:"} {
		section := schemaSection(t, body, name)
		if !strings.Contains(section, "additionalProperties: false") {
			t.Fatalf("%s accepts undocumented top-level fields", name)
		}
	}
}

func readOpenAPI(t *testing.T) string {
	t.Helper()
	body, err := os.ReadFile("../../docs/implementation/contracts/openapi-v1.yaml")
	if err != nil { t.Fatal(err) }
	return string(body)
}

func parseOperations(t *testing.T, body string) []string {
	t.Helper()
	var operations []string
	path := ""
	inPaths := false
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "paths:" { inPaths = true; continue }
		if inPaths && line == "components:" { break }
		if !inPaths { continue }
		if strings.HasPrefix(line, "  /") && strings.HasSuffix(line, ":") {
			path = strings.TrimSuffix(strings.TrimSpace(line), ":")
			continue
		}
		if path == "" || !strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "      ") { continue }
		method := strings.TrimSuffix(strings.TrimSpace(line), ":")
		switch method {
		case "get", "post", "put", "patch", "delete": operations = append(operations, strings.ToUpper(method)+" "+path)
		}
	}
	if err := scanner.Err(); err != nil { t.Fatal(err) }
	return operations
}

func pathSection(t *testing.T, body, marker string) string {
	t.Helper()
	start := strings.Index(body, marker)
	if start < 0 { t.Fatalf("path %s not found", marker) }
	rest := body[start+len(marker):]
	if next := strings.Index(rest, "\n  /"); next >= 0 { rest = rest[:next] }
	return rest
}

func schemaSection(t *testing.T, body, name string) string {
	t.Helper()
	marker := "    " + name
	start := strings.Index(body, marker)
	if start < 0 { t.Fatalf("schema %s not found", name) }
	rest := body[start+len(marker):]
	if next := strings.Index(rest, "\n    "); next >= 0 { rest = rest[:next] }
	return rest
}
