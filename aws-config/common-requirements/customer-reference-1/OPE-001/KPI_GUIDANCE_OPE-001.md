# Workload Health KPI Guidance (OPE-001)

This guidance is for customers operating the **AWS Config Cost Optimization Conformance Pack Project**.

## Purpose
Define, collect/analyze, and alert on workload health KPIs for the solution components. This guidance covers:
- Metrics (KPIs) and where to obtain them
- Logs required for troubleshooting
- Recommended alert thresholds to detect operational events

## Components in Scope
- CloudFormation StackSet (org-wide deployment health)
- AWS Config Organization Conformance Pack + member-account Config rules
- Lambda evaluator (`CostOptimizationConformanceConfigRuleFunction`)
- SSM Automation (remediation execution)

## 1) Metrics (Define, Collect, Analyze)

### KPI A — Compliance State / Drift Detection
- **Definition:** % COMPLIANT vs NON_COMPLIANT per rule (and overall conformance pack)
- **Collection:** AWS Config Conformance Packs / Rules (delegated admin visibility)
- **Analysis:** Trend compliance over time; prioritize the largest sources of noncompliance

### KPI B — Finding Volume
- **Definition:** count of NON_COMPLIANT resources per rule, account, and region
- **Collection:** AWS Config rule results
- **Analysis:** Identify spikes after deployments; identify accounts/teams creating new noncompliant resources

### KPI C — Evaluation Activity (Alert Volume / Signal Rate)
- **Definition:** evaluation executions over time (periodic + change-triggered)
- **Collection:** AWS Config evaluation activity + Lambda invocation volume
- **Analysis:** Compare to expected baseline (periodic evaluation) and flag abnormal bursts or gaps

### KPI D — Remediation Effectiveness
- **Definition:** remediation success rate and time-to-compliance
- **Collection:** SSM Automation execution status + follow-up Config re-evaluation
- **Analysis:** Track failure reasons (permissions, resource state prerequisites) and reduce repeat failures

### KPI E — Deployment Health (Multi-account Coverage)
- **Definition:** % of StackSet instances CURRENT across target OUs/regions
- **Collection:** CloudFormation StackSet operation/instance status
- **Analysis:** Any FAILED/OUTDATED instances indicate incomplete rollout or drift

## 2) Logs (Errors and Troubleshooting)

### Required log sources
- **Lambda evaluator logs:** CloudWatch Logs for `CostOptimizationConformanceConfigRuleFunction`
  - Use for: evaluation errors, timeouts, unexpected payloads
- **SSM Automation execution output:** Systems Manager Automation execution details
  - Use for: remediation failures, step output
- **CloudFormation / StackSet events:** CloudFormation Events for stack/StackSet operations
  - Use for: deployment failures, permissions issues, drift

### Ticket fields (recommended)
Record: account, region, rule name, resource ID, evaluation time, remediation execution ID (if used).

## 3) Thresholds (Alerting)

Recommended starter thresholds (adjust per customer criticality and scale):

- **Compliance state:** alert if any production rule has NON_COMPLIANT resources > 0 for > 24 hours (except documented exceptions).
- **Finding volume:** alert if NON_COMPLIANT count increases by > 20% week-over-week for any rule.
- **Lambda errors:** alert if Lambda `Errors` > 0 sustained for 15 minutes or error rate > 1%.
- **Remediation failures:** alert if Automation executions FAILED > 0/day or success rate < 95% over 7 days.
- **StackSet health:** alert if any StackSet instances are FAILED/OUTDATED for > 60 minutes.

Alert routing options:
- CloudWatch Alarms → SNS/Email/Ticketing
- EventBridge rules (service events) → notifications/ticketing
- Third-party observability: forward CloudWatch metrics/logs to SIEM/observability platform

## Evidence Location
- KPI guidance (this document): `ac-cp-co-main/docs/operations/KPI_GUIDANCE_OPE-001.md`
- KPI-driven operational runbook: `ac-cp-co-main/docs/operations/CUSTOMER_RUNBOOK_OPE-002.md`
- Rule definitions and evaluation frequency: `ac-cp-co-main/cdk-infra/lib/member-stack.ts`
- StackSet deployment model: `ac-cp-co-main/cdk-infra/lib/main-stack.ts`