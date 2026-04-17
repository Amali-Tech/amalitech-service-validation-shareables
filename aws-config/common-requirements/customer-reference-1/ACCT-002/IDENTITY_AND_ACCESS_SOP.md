# Identity and Access SOP (Customer Environment Access)

## Purpose
Define the standard, secure approach for how delivery engineers access customer-owned AWS environments during engagements.

This SOP is used for engagements delivered with the **AWS Config Cost Optimization Conformance Pack Project**.

## Scope
Applies to:
- AWS Management Console access
- Programmatic access (AWS CLI / SDK tools)
- Access across AWS Organizations (management, audit/delegated admin, and member accounts)

## Standard Access Approach

### 1) Human Access (Console and CLI)
**Policy:** All human access uses federated identity and MFA.

**Preferred implementation:**
- Use AWS IAM Identity Center (AWS SSO) for console and CLI access
- Federate IAM Identity Center to the customer enterprise identity provider (IdP) (e.g., SAML/OIDC)

**Console access:**
- Users sign in to the customer’s IAM Identity Center portal and select assigned accounts/roles

**Programmatic access:**
- Use AWS CLI SSO profiles (no long-lived access keys)

**Verification (evidence to capture):**
- Screenshot/export of IAM Identity Center user assignments showing per-user access
- CLI profile configuration demonstrating SSO-based access (no access keys)

### 2) Temporary Credentials (IAM Roles)
**Policy:** Use temporary credentials via IAM roles for all privileged access.

**Implementation guidance:**
- Use role sessions from IAM Identity Center (or STS AssumeRole) for admin tasks
- Do not create IAM users for engineers unless required by customer constraints
- For cross-account operations, use delegated administrator and service-managed mechanisms where available

**Verification (evidence to capture):**
- Evidence that access is role-based (role ARNs used; session-based access)

### 3) Leveraging Customer Enterprise Identities (Federation / Directory)
**Policy:** Customer enterprise identities are the source of truth.

**Implementation guidance:**
- Federate IAM Identity Center to customer IdP (SAML/OIDC) for workforce identities
- Where the customer requires Windows-integrated directory services for workloads, use AWS Managed Microsoft AD (AWS Directory Service)

**Verification (evidence to capture):**
- Architecture note or configuration evidence that IAM Identity Center is federated
- If applicable, evidence of AWS Managed Microsoft AD deployment/usage

## IAM Best Practices

### 4) Least Privilege and Minimizing Wildcards
**Policy:** Grant the minimum permissions required; avoid wildcards in `Action` and `Resource` where practical.

**Implementation guidance:**
- Prefer AWS managed policies for well-scoped service roles
- Use custom IAM policies with narrowly scoped actions
- Scope resources by ARN where supported; when wildcard is required, constrain via:
  - Conditions (tags, resource attributes)
  - Service-linked roles / service principals

**Verification (evidence to capture):**
- Role/policy review notes showing limited actions and principals

### 5) Dedicated Credentials (No Shared Accounts)
**Policy:** Every partner individual must use dedicated credentials.

**Implementation guidance:**
- Each engineer is provisioned as an individual IAM Identity Center user (or federated user) with MFA
- Do not share IAM users, access keys, or root credentials
- Use break-glass only for emergencies and log/approve usage

**Verification (evidence to capture):**
- IAM Identity Center user list (or IdP user list) showing unique identities
- Audit evidence (CloudTrail) for privileged actions when requested by customer

## How This SOP Is Applied in This Project (Customer Example)

The **AWS Config Cost Optimization Conformance Pack Project** is operated using a delegated administrator model:
- Management account actions are limited to enabling trusted service access and registering delegated administrators.
- Audit (delegated admin) account is used for centralized deployment and governance actions.
- Member accounts receive workloads through service-managed automation (StackSets) and service roles.

Project-aligned access patterns:
- Human access via IAM Identity Center/SSO (see `docs/getting_started/SETUP.md`)
- Multi-account deployment via service-managed CloudFormation StackSets (see `cdk-infra/lib/main-stack.ts`)
- Service roles used for AWS Config evaluation and SSM remediation in member accounts (see `cdk-infra/lib/member-stack.ts`)

## Evidence Location
- This SOP document (primary evidence): `ac-cp-co-main/docs/governance/IDENTITY_AND_ACCESS_SOP.md`
- Supporting project evidence:
  - `ac-cp-co-main/docs/getting_started/SETUP.md`
  - `ac-cp-co-main/docs/getting_started/PREREQUISITES.md`
  - `ac-cp-co-main/cdk-infra/lib/main-stack.ts`
  - `ac-cp-co-main/cdk-infra/lib/member-stack.ts`
