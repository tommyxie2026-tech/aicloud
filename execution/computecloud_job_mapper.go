package execution

import (
 "fmt"
 "strings"
)

type ComputecloudJobOptions struct {
 ProjectID, RepositoryRef, BaseCommit string
 Engine, Model, CredentialRef, PolicyRef, AcceptanceProfile string
}

func NewComputecloudJobMapper(o ComputecloudJobOptions) func(ExecutionRequest)(any,error) {
 return func(r ExecutionRequest)(any,error){
  if r.Goal==""{return nil,fmt.Errorf("execution goal is required")}
  if o.ProjectID==""||o.RepositoryRef==""||o.BaseCommit==""{return nil,fmt.Errorf("computecloud project, repository and base commit are required")}
  engine:=o.Engine;if engine==""{engine="codex"}
  runtime:="codex_exec";if engine=="claude"{runtime="claude_print"}else if engine!="codex"{return nil,fmt.Errorf("unsupported computecloud engine %q",engine)}
  timeout:=int64(r.Timeout.Seconds());if timeout<1{timeout=1800}
  policy:=o.PolicyRef;if policy==""{policy="aicloud-default"}
  if len(r.Permissions)>0{policy=policy+"-"+strings.Join(r.Permissions,",")}
  return map[string]any{
   "schema_version":"v0.2","project_id":o.ProjectID,"mode":"single",
   "workspace":map[string]any{"repository_ref":o.RepositoryRef,"base_commit":o.BaseCommit},
   "input":map[string]any{"text":r.Goal},
   "execution":map[string]any{"engine":engine,"runtime_profile":runtime,"model":o.Model,"credential_ref":o.CredentialRef,"policy_ref":policy,"acceptance_profile":o.AcceptanceProfile},
   "limits":map[string]any{"timeout_seconds":timeout,"max_attempts_per_task":1},
  },nil
 }
}
