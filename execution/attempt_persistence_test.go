package execution

import "testing"

func TestAttemptCompletionTerminalStates(t *testing.T) {
	for _, status := range []AttemptStatus{AttemptSucceeded, AttemptFailed, AttemptCancelled, AttemptAbandoned} {
		if !(AttemptCompletion{Status: status}).Terminal() {
			t.Fatalf("status %s should be terminal", status)
		}
	}
	for _, status := range []AttemptStatus{AttemptPending, AttemptRunning} {
		if (AttemptCompletion{Status: status}).Terminal() {
			t.Fatalf("status %s should not be terminal", status)
		}
	}
}
