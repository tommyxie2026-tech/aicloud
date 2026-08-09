# AI Cloud Resource Scope Matrix

| Resource | Scope |
|---|---|
| Provider | Global |
| Model | Global or Tenant |
| ModelVersion | Same as Model |
| Deployment | Global or Tenant |
| Agent | Project |
| Task | Project |
| TaskEvent | Task |
| RouteDecision | Task |
| ModelAttempt | Task |
| ToolInvocation | Task |
| Policy | Global/Tenant/Project |
| EvaluationDataset | Global/Tenant |
| EvaluationRun | Task/Project |
| CostEvent | Task |
| AuditEvent | Tenant + Project |
| CredentialGrant | Task |
| Approval | Task |

## Rule

Every resource must have exactly one ownership boundary. Cross-boundary access must be explicit through policy evaluation.

Global resources cannot contain tenant runtime state.
Tenant resources cannot silently access another tenant.
Task resources inherit Task ownership.
