package execution

import "fmt"

var executionTransitions = map[ExecutionPhase]map[ExecutionPhase]struct{}{
	ExecutionCreated: {
		ExecutionPlanning:  {},
		ExecutionCancelled: {},
	},
	ExecutionPlanning: {
		ExecutionReady:     {},
		ExecutionFailed:    {},
		ExecutionCancelled: {},
	},
	ExecutionReady: {
		ExecutionRunning:   {},
		ExecutionFailed:    {},
		ExecutionCancelled: {},
	},
	ExecutionRunning: {
		ExecutionVerifying:  {},
		ExecutionReplanning: {},
		ExecutionFailed:     {},
		ExecutionCancelled:  {},
	},
	ExecutionVerifying: {
		ExecutionSucceeded:  {},
		ExecutionReplanning: {},
		ExecutionFailed:     {},
		ExecutionCancelled:  {},
	},
	ExecutionReplanning: {
		ExecutionReady:     {},
		ExecutionFailed:    {},
		ExecutionCancelled: {},
	},
}

func CanTransitionExecution(from, to ExecutionPhase) bool {
	if from == to {
		return true
	}
	next, ok := executionTransitions[from]
	if !ok {
		return false
	}
	_, ok = next[to]
	return ok
}

func ValidateExecutionTransition(from, to ExecutionPhase) error {
	if CanTransitionExecution(from, to) {
		return nil
	}
	return fmt.Errorf("invalid execution transition: %s -> %s", from, to)
}

func IsTerminalExecutionPhase(phase ExecutionPhase) bool {
	switch phase {
	case ExecutionSucceeded, ExecutionFailed, ExecutionCancelled:
		return true
	default:
		return false
	}
}

var nodeTransitions = map[NodeState]map[NodeState]struct{}{
	NodePending: {
		NodeReady:     {},
		NodeSkipped:   {},
		NodeBlocked:   {},
		NodeCancelled: {},
	},
	NodeReady: {
		NodeRunning:         {},
		NodeWaitingApproval: {},
		NodeBlocked:         {},
		NodeCancelled:       {},
	},
	NodeWaitingApproval: {
		NodeReady:     {},
		NodeBlocked:   {},
		NodeCancelled: {},
	},
	NodeRunning: {
		NodeSucceeded: {},
		NodeFailed:    {},
		NodeCancelled: {},
	},
	NodeFailed: {
		// A retry is represented by a new Attempt. The Node itself may be made
		// READY again only when retry policy and Supervisor allow it.
		NodeReady:     {},
		NodeBlocked:   {},
		NodeCancelled: {},
	},
	NodeBlocked: {
		NodeReady:     {},
		NodeCancelled: {},
	},
}

func CanTransitionNode(from, to NodeState) bool {
	if from == to {
		return true
	}
	next, ok := nodeTransitions[from]
	if !ok {
		return false
	}
	_, ok = next[to]
	return ok
}

func ValidateNodeTransition(from, to NodeState) error {
	if CanTransitionNode(from, to) {
		return nil
	}
	return fmt.Errorf("invalid node transition: %s -> %s", from, to)
}

func IsTerminalNodeState(state NodeState) bool {
	switch state {
	case NodeSucceeded, NodeSkipped, NodeCancelled:
		return true
	default:
		return false
	}
}
