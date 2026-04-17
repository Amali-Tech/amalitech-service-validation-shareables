# Deployment Automation (REL-001)

This document describes how deployments are automated for the **AWS Config Cost Optimization Conformance Pack Project**.

## Deployment Automation Approach

### Infrastructure as Code (IaC)
The project provisions and updates infrastructure using Infrastructure-as-Code:
- **AWS CDK (TypeScript)** synthesizes CloudFormation templates and deploys them via the CLI.
- A packaged **CloudFormation/SAM template** is provided for customer deployment (`template/template.yaml`) which provisions:
  - Organization-wide CloudFormation StackSet (SERVICE_MANAGED, delegated admin)
  - AWS Config Organization Conformance Pack
  - Supporting Lambda custom resources (org details lookup, document sharing)

### Deployment is Automated (CLI / Pipeline)
Standard deployment is executed using automation tooling:
- Local automation via repeatable commands (build/synth/deploy) in `cdk-infra/`.
- CI automation runs build + tests on pull requests and on the main branch.

### Console Usage Policy
For customer implementations, **infrastructure changes are applied via IaC (CloudFormation/CDK) and CLI automation**.
- AWS Management Console is used for **read-only verification** (checking stack status, conformance pack status) and, where required by customer operations, to **trigger remediation executions**.
- Baseline infrastructure deployment and updates (stacks, StackSets, conformance packs, roles, functions, documents) are not performed manually in the console.

## Example Templates / Artifacts
- CloudFormation/SAM deployment template (example template evidence): `template/template.yaml`
- CDK implementation (source IaC): `cdk-infra/lib/main-stack.ts`, `cdk-infra/lib/member-stack.ts`

## Supporting Automation Evidence
- Deployment guide (CLI-based): `cdk-infra/DEPLOYMENT.md`
- Automated tests (CI pipeline): `.github/workflows/test.yml`
- Local automation scripts: `cdk-infra/package.json`
