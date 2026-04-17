# Customer Runbook / Playbook (OPE-002)

This runbook is for customers operating the **AWS Config Cost Optimization Conformance Pack Project** across an AWS Organization.

## Purpose
Provide standardized operational tasks and troubleshooting scenarios that directly support the workload health KPIs defined in OPE-001.

## Scope
Covers:
- AWS Config conformance pack/rule health and compliance monitoring
- Operational triage and remediation workflows
- Multi-account deployment health (StackSet) and common failure scenarios

## OPE-001 KPIs and Operational Signals
The project operationally tracks OPE-001 KPIs using AWS native service signals. KPI definitions and recommended thresholds are standardized in:

- `docs/operations/KPI_GUIDANCE_OPE-001.md`

KPI signals used by this runbook:

1. **Compliance State / Drift Detection**
   - Signal: AWS Config conformance pack and rule compliance status
2. **Finding Volume**
   - Signal: number of NON_COMPLIANT resources per rule, per account/region
3. **Evaluation Activity (Alert Volume / Signal Rate)**
   - Signal: evaluation cadence (periodic + change-triggered) and spikes in evaluations
4. **Remediation Effectiveness**
   - Signal: SSM Automation execution success/failure and time-to-compliance after remediation
5. **Deployment Health (Multi-account Coverage)**
   - Signal: StackSet instance status (CURRENT/FAILED/OUTDATED) across target OUs/regions

## Routine Operational Tasks

### Daily
- Review AWS Config conformance pack compliance in the delegated administrator (audit) account.
- Triage new NON_COMPLIANT resources for the three rules:
  - `CostOpt-Ebs-Gp3`
  - `CostOpt-Ebs-Unattached`
  - `CostOpt-S3-WithoutLifecycle`
- For actionable items, remediate and confirm re-evaluation returns COMPLIANT.

### Weekly
- Review trend: finding volume per rule (is it trending down?).
- Validate multi-account deployment health:
  - StackSet operation status and StackSet instances are CURRENT.
- Review Lambda execution logs for errors in the rule evaluator function.

### Monthly / Release-based
- Validate runbook assumptions (evaluation frequency, remediation enablement) match customer operational policy.
- Review IAM permissions for remediation roles to ensure least-privilege remains effective.

## Standard Triage Flow (All Scenarios)
1. **Identify**: rule name, account/region, and resource.
2. **Validate**: confirm noncompliance is actionable (not an intentional exception).
3. **Remediate**:
   - Prefer SSM Automation where configured for the rule.
   - Otherwise apply manual remediation.
4. **Verify**: re-evaluate rule and confirm COMPLIANT.
5. **Record**: capture evidence for the ticket (resource ID, timestamps, remediation execution ID if used).

## Troubleshooting Scenarios

### Scenario A — Rule is not evaluating / results are stale
**Symptoms**: No recent evaluations; compliance status not updating.

**Checks**:
- AWS Config recorder is enabled and delivery channel is healthy in the member account.
- Lambda function has no recent errors.

**Actions**:
- Trigger manual re-evaluation for the rule.
- Review CloudWatch logs for the evaluator function.

### Scenario B — Remediation failed
**Symptoms**: SSM Automation execution FAILED or resource remains NON_COMPLIANT.

**Checks**:
- SSM Automation execution output for error reason.
- IAM permissions on the Automation role (e.g., `ec2:ModifyVolume` for gp3 conversion).
- Resource state prerequisites (e.g., volume state/snapshot operations).

**Actions**:
- Correct permission/state issue and retry remediation.
- If needed, apply manual remediation and re-evaluate.

### Scenario C — StackSet deployment drift / member account missing rules
**Symptoms**: Some accounts/regions do not show the conformance pack/rules.

**Checks**:
- StackSet instances status (OUTDATED/FAILED).
- Target OU and region list used for deployment.

**Actions**:
- Re-run StackSet operation and validate instances become CURRENT.
- Confirm trusted access and delegated administrator settings remain configured.

### Scenario D — Excessive alert volume or unexpected finding spike
**Symptoms**: sudden increase in evaluations or NON_COMPLIANT resources.

**Checks**:
- Recent infrastructure changes (new accounts, new regions, change in evaluation frequency).
- Recent workload deployments creating gp2 volumes or buckets without lifecycle.

**Actions**:
- Identify the top contributing accounts/resources.
- Apply targeted remediation and communicate required guardrails to workload teams.

## Project Evidence References
This runbook is supported by these project documents and implementations:
- Deployment + operational walkthrough (console steps for re-evaluation and remediation): `docs/getting_started/README.md`
- Deployment prerequisites (delegated admin + trusted access model): `docs/getting_started/PREREQUISITES.md`
- Testing guidance for creating noncompliant resources and validating evaluations: `cdk-infra/TESTING.md`
- Deployment procedure: `cdk-infra/DEPLOYMENT.md`
- Rule definitions and scopes: `cdk-infra/lib/member-stack.ts`
- Remediation documents definition: `cdk-infra/lib/constructs/remediation-documents.ts`
