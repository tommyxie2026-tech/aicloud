package execution

import "testing"

func TestExecutionAndOutcomeAreSeparate(t *testing.T) {
	if string(ExecutionCompleted) == string(OutcomeVerified) {
		t.Fatal("provider completion must not equal verified outcome")
	}
}
