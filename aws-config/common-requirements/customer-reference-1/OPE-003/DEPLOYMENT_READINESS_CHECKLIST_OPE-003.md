# Deployment Readiness Checklist (OPE-003)

This checklist is for deployments of the **AWS Config Cost Optimization Conformance Pack Project**.

## Purpose
Provide a consistent readiness gate to ensure deployments are validated before production and include automated testing components.

## Checklist

### A. Pre-Deployment (Environment Readiness)
- Confirm AWS Organizations + Control Tower is in place and the audit/delegated admin account is identified.
- Confirm trusted access is enabled and delegated administrators are registered for:
  - `config.amazonaws.com`
  - `config-multiaccountsetup.amazonaws.com`
  - `member.org.stacksets.cloudformation.amazonaws.com`
- Confirm the deployment operator has access to the management account (for org setup) and audit account (for deployment).
- Confirm AWS SSO/IAM Identity Center access is configured for the audit profile.

### B. Code & Configuration Validation
- Confirm `.env` (or environment variables) are configured for `CDK_DEPLOY_ACCOUNT`, `CDK_DEPLOY_REGION`, and `AWS_PROFILE`.
- Run dependency install:
  - `cdk-infra/`: `npm ci` (preferred) or `npm install`
- Build TypeScript:
  - `npm run build`

### C. Automated Tests (Gate)
- Run automated TypeScript (CDK) tests:
  - `npm test` (or `npm run test:coverage`)
- Run automated Python (Lambda evaluator) tests:
  - `npm run test:python` (or `npm run test:python:coverage`)

**CI expectation:** these tests must pass in the repository workflow before merging to `main`.

### D. Template Validation (Pre-Production)
- Synthesize templates to detect errors early:
  - `npm run synth`
- Review change set / drift risk:
  - `npm run diff`

### E. Controlled Rollout
- Deploy to a non-production OU or a small pilot OU first where possible.
- Deploy main stack:
  - `npm run deploy:main` (or `cdk deploy CostOptimizationMainStack` with the audit profile)
- Verify StackSet instance status is CURRENT in target accounts/regions.

### F. Post-Deployment Verification (Production Readiness)
- Verify conformance pack status and rule presence in member accounts.
- Execute a functional test:
  - Create a known noncompliant resource (e.g., an EBS gp2 volume) and verify evaluation.
  - Trigger remediation for gp2 → gp3 (where enabled) and confirm compliance returns to COMPLIANT.
- Capture evidence for the deployment ticket:
  - Stack/StackSet operation status
  - Conformance pack/rule status
  - Test evaluation + remediation execution (if applicable)

## Supporting Project Evidence
- Deployment procedure: `cdk-infra/DEPLOYMENT.md`
- Testing strategy and how to run tests: `cdk-infra/TESTING.md`
- Automated test pipeline: `.github/workflows/test.yml`
- Local automated test scripts: `cdk-infra/package.json`
- Console testing steps (re-evaluate and remediate): `docs/getting_started/README.md`
