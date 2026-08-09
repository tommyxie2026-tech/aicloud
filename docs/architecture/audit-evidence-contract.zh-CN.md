# AI Cloud Audit Evidence 契约

> 状态：S0 Contract Freeze

## 1. 目标

定义 Identity、Policy、Execution、Governance Decision 的不可变 Audit Evidence。Audit 是业务/安全证据，不是 Application Log 的复制品。

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

AuditEvent 必须 Append-only。

## 3. Mandatory Audit Event

至少记录：

```text
Authentication Failure/Success（按 Security Policy 要求）
Protected Resource Authorization Deny
System Principal 使用
Task Create/Cancel/Recovery
High-risk PolicyDecision
Approval Grant/Reject/Revoke
Model Admission/Promotion/Rollback
Tool Invocation Requested/Executed/Failed
Credential Grant Issue/Revoke
Administrative DB/Tenant Operation
Governance Configuration Change
```

## 4. Audit / Trace / TaskEvent 区分

```text
TaskEvent = Task 的 Business History
Trace     = Execution Telemetry
Audit     = Security/Governance Evidence
```

同一次真实操作可以同时产生三类 Record，通过 Task/Trace ID 关联，但 Retention 与 Access Policy 不同。

## 5. Payload Rule

Audit 尽量保存 Evidence Reference 与 Digest，不存储 Raw Secret、Credential 或不必要的敏感 Prompt/Data Payload。

High-risk Proposal 应保存 Immutable Digest + Governed Artifact Reference，而不是只保存可读文本。

## 6. Integrity

普通 Application API 不允许修改 Audit Record。生产环境应根据 Assurance Level 使用 Append-only Storage、Restricted DB Role、Hash Chaining、Export to Immutable Storage 或等价 Tamper-evident Control。

## 7. Reason Code

Security/Governance Decision 使用稳定 Machine-readable Reason Code，并可附加安全的人类说明。这样既支持 Reporting，也减少泄露 Hidden Policy 或 Cross-tenant Information 的风险。

## 8. Access Control

Audit Access 必须 Tenant-aware 且 Role-restricted。Platform Operator 不自动拥有所有 Tenant Sensitive Payload Reference 的无限访问权。

## 9. Retention

Retention 由 Policy/Legal Requirement 决定，通常长于 Application Log。Deletion/Compaction 必须通过受治理 Retention Process，并对删除本身产生 Audit Evidence。

## 10. Correlation

Task-related Audit Event 必须至少包含：

```text
tenant_id
project_id
task_id
trace_id
principal
resource/action
```

非 Task 的 Admin Event 在适用时仍必须带 Tenant/Project，并保留 Trace/Request Correlation Key。

## 11. Failure Semantics

High-risk Mutation 如果无法持久化 Mandatory Audit Evidence，应 Fail Closed，除非已有正式批准的 Buffered/Immutable Fallback。Audit Loss 不能静默忽略。

## 12. 验收条件

- Mandatory Action 的 Allow/Deny 路径都产生 Audit Event；
- 普通 Application API 不能 Update/Delete Audit Event；
- Audit Payload 不包含 Secret；
- System Principal Action 全部记录；
- Audit 可关联 Task/Trace/Policy/Approval Evidence；
- Tenant-aware Audit Access Test 通过；
- Audit Store Outage 时 High-risk Operation 行为确定且有文档。