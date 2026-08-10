# AI Cloud Audit Evidence Contract

> Status: S0 Contract Freeze

## 1. Purpose

Define immutable audit evidence for identity, policy, execution and governance decisions. Audit is a business/security record, not a copy of application logs.

## 2. AuditEvent

```yaml
audit_event:
  audit_event_id: string
  tenant_id: string
  project_id: string
  task_id: string?
  trace_id: string
  request_id: string?
  principal:
    type: string
    subject_id: string
  action: string
  resource_type: string
  resource_id: string
  decision: allow | deny | attempted | succeeded | failed
  reason_codes: []string
  policy_decision_id: string?
  approval_id: string?
  evidence_refs: []string
  metadata: object
  occurred_at: timestamp
  created_at: timestamp
```

AuditEvent is append-only.

## 3. Mandatory Audit Events

At minimum record:

```text
Authentication failure/success where required by security policy
Authorization deny for protected resources
System Principal use
Task creation/cancel/recovery
High-risk PolicyDecision
Approval grant/reject/revoke
Model admission/promotion/rollback
Tool invocation requested/executed/failed
Credential grant issuance/revocation
Administrative DB/tenant operation
Governance configuration changes
```

## 4. Audit vs Trace vs TaskEvent

```text
TaskEvent = business history of Task
Trace     = execution telemetry
Audit     = security/governance evidence
```

One real-world action may create records in all three, linked by Task/Trace IDs. These stores have different retention and access policies.

## 5. Payload Rules

Audit stores evidence references and digests where possible. It must not store raw secrets, credentials or unnecessary sensitive prompt/data payloads.

For high-risk proposals, store an immutable digest and governed artifact reference rather than only human-readable text.

## 6. Integrity

Audit records must not be editable through ordinary application APIs. Production deployments should support tamper-evident controls such as append-only storage, restricted DB roles, hash chaining/export to immutable storage, or equivalent controls according to assurance level.

## 7. Reason Codes

Security/governance decisions use stable machine-readable reason codes plus optional safe human explanation. This supports reporting and avoids leaking hidden policy or cross-tenant information.

## 8. Access Control

Audit access is tenant-aware and role-restricted. Platform operators do not automatically receive unrestricted access to sensitive tenant payload references.

## 9. Retention

Retention is policy/legal driven and usually longer than application logs. Deletion/compaction requires a governed retention process and its own audit evidence.

## 10. Correlation

A Task-related audit event must include:

```text
tenant_id
project_id
task_id
trace_id
principal
resource/action
```

A non-Task administrative event still requires tenant/project where applicable and a trace/request correlation key.

## 11. Failure Semantics

For high-risk mutations, inability to persist mandatory audit evidence should fail closed unless a formally approved buffered/immutable fallback exists. Audit loss must never be silently ignored.

## 12. Acceptance Criteria

- Mandatory actions produce audit events for both allow and deny paths.
- Ordinary application APIs cannot update/delete audit events.
- Secrets are not present in audit payloads.
- System Principal actions are always recorded.
- Audit records correlate to Task/Trace/Policy/Approval evidence.
- Tenant-aware audit access tests pass.
- High-risk operation behavior under audit-store outage is deterministic and documented.