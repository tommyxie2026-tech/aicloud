package authorization

import "strings"

const (
	RoleTenantAdmin = "tenant_admin"
	RoleProjectAdmin = "project_admin"
	RoleDeveloper = "developer"
	RoleOperator = "operator"
	RoleReviewer = "reviewer"
	RoleViewer = "viewer"
)

type RBAC struct { Version string }

func BuiltinRBAC() *RBAC { return &RBAC{Version: "builtin-rbac-v1"} }

func (r *RBAC) Evaluate(req Request) Decision {
	version := "builtin-rbac-v1"
	if r != nil && r.Version != "" { version = r.Version }
	if r == nil { return Decision{Layer: "rbac", PolicyVersion: version, Reason: "role evaluator is not configured"} }
	for _, value := range req.Principal.Roles {
		role := strings.ToLower(strings.TrimSpace(value))
		if roleAllows(role, req.Action) { return Decision{Allowed: true, Layer: "rbac", PolicyVersion: version, MatchedRole: role, Reason: "role grants action"} }
	}
	return Decision{Layer: "rbac", PolicyVersion: version, Reason: "role does not grant action"}
}

func roleAllows(role string, action Action) bool {
	if role == RoleTenantAdmin { return true }
	if isReadAction(action) { return role == RoleProjectAdmin || role == RoleDeveloper || role == RoleOperator || role == RoleReviewer || role == RoleViewer }
	switch action {
	case ActionTaskCreate, ActionTaskRoute, ActionTaskModelExecute, ActionToolExecute:
		return role == RoleProjectAdmin || role == RoleDeveloper || role == RoleOperator
	case ActionTaskEvaluationWrite:
		return role == RoleProjectAdmin || role == RoleReviewer
	default:
		return false
	}
}

func isReadAction(action Action) bool {
	switch action {
	case ActionModelRead, ActionModelAdmissionRead, ActionToolRead, ActionTaskRead, ActionTaskRouteRead, ActionTaskCostRead, ActionTaskAuditRead, ActionTaskTraceRead, ActionTaskEvaluationRead:
		return true
	default:
		return false
	}
}
