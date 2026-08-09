# ADR-0018: AI Enterprise Operating Model

## Status
Accepted

## Context

AI Cloud has evolved from a model gateway into an enterprise AI operating platform. Technical capabilities alone are insufficient; organizations need operating models covering governance, ownership, lifecycle management, and business adoption.

## Decision

Introduce an AI Enterprise Operating Model layer covering:

- AI Center of Excellence
- Business AI Teams
- Governance Board
- AI Service Catalog
- Chargeback and FinOps
- Risk Management

## Operating Model

```text
Enterprise AI Operating Model

        AI Governance Board
                 |
                 v
       AI Center of Excellence
                 |
     +-----------+-----------+
     |                       |
Business AI Teams     Platform Team
     |                       |
AI Applications       AI Cloud Platform
```

## Principles

1. AI governance must be centralized, execution must be decentralized.
2. AI assets must have ownership and lifecycle management.
3. Production AI requires evaluation, security, and cost controls.
4. Business value is measured by outcomes, not model usage.

## Impact

AI Cloud becomes not only a technical platform but an enterprise AI operating system enabling scalable adoption.
