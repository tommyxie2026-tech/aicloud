package execution

import "fmt"

// ValidatePlan enforces the R0 bounded-DAG invariants.
func ValidatePlan(plan ExecutionPlan) error {
	if plan.ID == "" {
		return fmt.Errorf("plan id is required")
	}
	if plan.GoalRef == "" {
		return fmt.Errorf("goal ref is required")
	}
	if plan.Revision <= 0 {
		return fmt.Errorf("plan revision must be positive")
	}
	if len(plan.Nodes) == 0 {
		return fmt.Errorf("plan must contain at least one node")
	}

	nodes := make(map[NodeID]ExecutionNode, len(plan.Nodes))
	for _, node := range plan.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node id is required")
		}
		if _, exists := nodes[node.ID]; exists {
			return fmt.Errorf("duplicate node id: %s", node.ID)
		}
		if err := ValidateEffectSpec(node.Effect); err != nil {
			return fmt.Errorf("node %s: %w", node.ID, err)
		}
		nodes[node.ID] = node
	}

	for _, node := range plan.Nodes {
		seenDeps := make(map[NodeID]struct{}, len(node.DependsOn))
		for _, dep := range node.DependsOn {
			if dep == node.ID {
				return fmt.Errorf("node %s depends on itself", node.ID)
			}
			if _, ok := nodes[dep]; !ok {
				return fmt.Errorf("node %s references unknown dependency %s", node.ID, dep)
			}
			if _, exists := seenDeps[dep]; exists {
				return fmt.Errorf("node %s contains duplicate dependency %s", node.ID, dep)
			}
			seenDeps[dep] = struct{}{}
		}
	}

	if hasCycle(plan.Nodes) {
		return fmt.Errorf("execution plan contains a cycle")
	}

	return nil
}

func ValidateEffectSpec(effect EffectSpec) error {
	switch effect.Class {
	case EffectPure, EffectReadOnly:
		return nil
	case EffectIdempotentMutation:
		if effect.IdempotencyKey == "" {
			return fmt.Errorf("idempotent mutation requires idempotency key")
		}
		if !effect.RetrySafe {
			return fmt.Errorf("idempotent mutation must declare retrySafe=true")
		}
		return nil
	case EffectNonIdempotentMutation:
		if effect.RetrySafe {
			return fmt.Errorf("non-idempotent mutation cannot declare retrySafe=true")
		}
		return nil
	default:
		return fmt.Errorf("unknown effect class %q", effect.Class)
	}
}

func hasCycle(nodes []ExecutionNode) bool {
	deps := make(map[NodeID][]NodeID, len(nodes))
	for _, node := range nodes {
		deps[node.ID] = append([]NodeID(nil), node.DependsOn...)
	}

	const (
		unseen = iota
		visiting
		done
	)
	state := make(map[NodeID]int, len(nodes))

	var visit func(NodeID) bool
	visit = func(id NodeID) bool {
		switch state[id] {
		case visiting:
			return true
		case done:
			return false
		}
		state[id] = visiting
		for _, dep := range deps[id] {
			if visit(dep) {
				return true
			}
		}
		state[id] = done
		return false
	}

	for id := range deps {
		if state[id] == unseen && visit(id) {
			return true
		}
	}
	return false
}
