# Workload Resilience Guidance (REL-002)

This guidance is for customer engagements delivered with the **AWS Config Cost Optimization Conformance Pack Project**.

## Purpose
Provide a consistent process to discuss resilience with customers and recommend Recovery Time Objective (RTO) and Recovery Point Objective (RPO) targets.

Customer acceptance/adoption of the recommended RTO/RPO is not required; the goal is to ensure awareness and a documented recommendation.

## 1) RTO/RPO Recommendation Process

### Step A — Classify criticality
For the workload in scope (either the customer workload or the governance tooling we deploy), capture:
- Impact if unavailable (audit/compliance visibility loss vs. business outage)
- Acceptable window of degraded monitoring
- Data retention needs for audit/compliance

### Step B — Propose targets (recommendation)
Use a simple tiering model and tailor to the customer:

- **Tier 1 (Standard)**: RTO ≤ 4 hours, RPO ≤ 24 hours
- **Tier 2 (Enhanced)**: RTO ≤ 1 hour, RPO ≤ 4 hours
- **Tier 3 (Critical)**: RTO ≤ 15 minutes, RPO ≤ 1 hour

### Step C — Record and communicate
Document the proposed RTO/RPO in the engagement deliverables (HLD/LLD/runbook) and communicate:
- What the targets apply to (governance tooling vs. customer application workloads)
- What technical controls are in place (and what is not implemented by default)
- Any exceptions and customer-owned dependencies

## 2) Recommended Targets for This Project (Governance Workload)

The **AWS Config Cost Optimization Conformance Pack Project** is a governance/monitoring solution. It is designed to be redeployable via IaC and does not host customer business transactions.

Recommended default targets for the governance solution (unless the customer requires stricter targets):
- **RTO**: ≤ 60 minutes (redeploy and restore monitoring/visibility)
- **RPO**: ≤ 24 hours for compliance history continuity (acceptable gap for most governance reporting); note that RPO for “configuration” is effectively 0 because the desired state is stored in version-controlled templates.

## 3) Recovery Process for Core Components

### Core components
- Delegated admin (audit) account main stack
- Organization-wide StackSet controller and member-account stacks
- AWS Config Organization Conformance Pack
- SSM Automation document(s) for remediation
- AWS Config delivery S3 buckets in member accounts (data retention)

### Recovery approach

1. **Redeploy the baseline (IaC rehydration)**
   - Use the repeatable CloudFormation/CDK deployment process to recreate the main stack.
   - StackSet re-syncs member-account resources.
   - Organization conformance pack is re-applied.

2. **Recover from drift or partial failures**
   - If StackSet state is inconsistent, recreate StackSet instances and re-run the update.
   - If the conformance pack drifts, delete and recreate it via stack update.

3. **Restore monitoring signal**
   - Verify conformance pack status and rule evaluations resume.
   - Run a functional test (create a known noncompliant resource and confirm evaluation).

### Evidence-backed recovery scenarios
The project architecture analysis includes recovery scenarios for accidental stack deletion, StackSet corruption, and conformance pack drift, along with recovery steps and an estimated recovery time.

## 4) Customer Awareness and Communication

Minimum engagement communication steps:
- Include “Resilience (RTO/RPO recommendation)” as a required agenda item in architecture review / kickoff.
- Provide the recommended targets and the recovery process in a customer-facing runbook.
- Clearly call out what is not implemented by default (for example, cross-region replication of AWS Config delivery data) and offer options if stricter RPO is required.

## Evidence Location
- This resilience guidance (primary evidence): `ac-cp-co-main/docs/operations/RESILIENCE_GUIDANCE_REL-002.md`
- DR scenarios and recovery steps: `ac-cp-co-main/template/ARCHITECTURE_ANALYSIS.md` (Disaster Recovery section)
- Repeatable deployment process: `ac-cp-co-main/cdk-infra/DEPLOYMENT.md`
- Member-account data retention behavior (S3 RETAIN + lifecycle): `ac-cp-co-main/cdk-infra/lib/member-stack.ts`
