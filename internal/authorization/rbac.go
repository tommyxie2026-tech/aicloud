package authorization

import "strings"

const (
	RoleTenantAdmin  = "tenant_admin"
	RoleProjectAdmin = "project_admin"
	RoleDeveloper    = "developer"
	RoleOperator     = "operator"
	RoleReviewer     = "reviewer"
	RoleViewer       = "viewer"
)

type RBAC struct {
	Version string
	grants  map[string]map[Action]struct{}
}

func NewRBAC(version string, grants map[string][]Action) *RBAC {
	compiled := make(map[string]map[Action]struct{}, len(grants))
	for role, actions := range grants {
		role = strings.ToLower(strings.TrimSpace(role))
		if role == "" {
			continue
		}
		set := make(map[Action]struct{}, len(actions))
		for _, action := range actions {
			if strings.TrimSpace(string(action)) != "" {
				set[action] = struct{}{}
			}
		compiled[role] = set
	}
	return &RBAC{Version: strings.TrimSpace(version), grants: compiled}
}

func BuiltinRBAC() *RBAC {
	readOnly := []Action{
		ActionModelRead,
		ActionModelAdmissionRead,
		ActionToolRead,
		ActionTaskRead,
		ActionTaskRouteRead,
		ActionTaskCostRead,
		ActionTaskAuditRead,
		ActionTaskTraceRead,
		ActionTaskEvaluationRead,
	}
	projectMutation := []Action{
		ActionTaskCreate,
		ActionTaskRoute,
		ActionTaskModelExecute,
		ActionToolExecute,
	}
	return NewRBAC("builtin-rbac-v1", map[string][]Action{
		RoleTenantAdmin: {ActionAny},
		RoleProjectAdmin: append(append([]Action{}, readOnly...), append(projectMutation, ActionTaskEvaluationWrite)...),
		RoleDeveloper: append(append([]Action{}, readOnly...), projectMutation...),
		RoleOperator: append(append([]Action{}, readOnly...), projectMutation...),
		RoleReviewer: append(append([]Action{}, readOnly...), ActionTaskEvaluationWrite),
		RoleViewer: append([]Action{}, readOnly...),
	})
}

func (r *RBAC) Evaluate(request Request) Decision {
	version := "rbac-unconfigured"
	if r != nil && r.Version != "" {
		version = r.Version
	}
	if r == nil {
		return Decision{Allowed: false, Reason: "RBAC is not configured", Layer: "rbac", PolicyVersion: version}
	}
	if request.Principal.Type == "system" {
		return Decision{Allowed: false, Reason: "system principals require a dedicated internal authorization path", Layer: "rbac", PolicyVersion: version}
	}
	for _, rawRole := range request.Principal.Roles {
		role := strings.ToLower(strings.TrimSpace(rawRole))
		permissions, ok := r.grants[role]
		if !ok {
			continue
		}
		if _, ok := permissions[ActionAny]; ok {
			return Decision{Allowed: true, Reason: "role grants all API actions", Layer: "rbac", PolicyVersion: version, MatchedRole: role}
		}
		if _, ok := permissions[request.Action]; ok {
			return Decision{Allowed: true, Reason: "role grants requested API action", Layer: "rbac", PolicyVersion: version, MatchedRole: role}
		}
	}
	return Decision{Allowed: false, Reason: "no principal role grants the requested API action", Layer: "rbac", PolicyVersion: version}
}
