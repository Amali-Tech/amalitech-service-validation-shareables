# Secure AWS Account Governance SOP

## Purpose
Define the minimum secure AWS account governance controls that the delivery team must apply when creating or onboarding AWS accounts for customers.

This SOP is used for engagements delivered with the **AWS Config Cost Optimization Conformance Pack Project**.

## Scope
Applies to:
- New AWS accounts created for a customer (via AWS Control Tower Account Factory)
- Existing customer AWS accounts onboarded into an AWS Organization with AWS Control Tower

Out of scope:
- Daily workload operations (must not use root credentials)

## Governance Requirements and Procedure

### 1) Root Account Usage Policy (Workload Activities)
**Policy:** The root user must not be used for workload activities.

**Allowed root usage (break-glass only):**
- Initial AWS Control Tower / Organization bootstrap if customer mandates root usage
- Account recovery (lost IAM Identity Center / admin access)
- Enabling/maintaining support plan and billing-level settings that require root

**Standard operating model:**
- Use AWS IAM Identity Center (AWS SSO) with MFA for human access
- Use delegated administrator accounts for service administration
- Use IAM roles for automation and service-to-service access

**Verification (evidence to capture):**
- Screenshot or exported report showing IAM Identity Center is the primary access path
- Confirmation that no workloads are operated using the root user

### 2) Root MFA
**Policy:** MFA is required on the root user for every account.

**Implementation guidance:**
- Enable MFA on the root user immediately after account creation/onboarding
- Use a hardware MFA device where required by the customer; otherwise use a virtual MFA
- Store break-glass instructions in the customer’s secure vault / password manager

**Verification (evidence to capture):**
- Console screenshot: “MFA enabled” for the root user

### 3) Corporate Contact Information
**Policy:** Account contact information must be corporate-owned and supportable.

**Implementation guidance (minimum):**
- Root account email: corporate distribution list (not a personal email)
- Phone number: corporate/service desk number
- Configure alternate contacts (Billing, Operations, Security) with corporate emails

**Verification (evidence to capture):**
- Console screenshots or exported contact details showing corporate email/phone

### 4) CloudTrail Logging (All Regions) and Log Protection
**Policy:** CloudTrail must be enabled for all regions and CloudTrail logs must be protected from accidental deletion using a dedicated S3 bucket.

**Implementation guidance:**
- Use AWS Control Tower’s baseline (recommended):
  - Organization trail enabled
  - Centralized delivery to the Log Archive account’s dedicated S3 log bucket
- Ensure:
  - Multi-region trail enabled
  - Global service events enabled
  - Log file validation enabled
  - S3 bucket protected with least privilege bucket policies
  - S3 versioning enabled (minimum)
  - If customer requires stronger controls: enable S3 Object Lock (governance/compliance mode) for WORM retention

**Verification (evidence to capture):**
- CloudTrail: organization trail is enabled and multi-region
- S3: log bucket has versioning enabled and delete is restricted

## How This SOP Is Applied in This Project
The **AWS Config Cost Optimization Conformance Pack Project** assumes a Control Tower governed AWS Organization and uses delegated administration and automation (not root credentials) to deploy AWS Config resources.

Operational model used by the project:
- Management account: only used to enable trusted access and register delegated administrators
- Audit (delegated admin) account: runs organization-wide deployment and hosts shared SSM automation documents
- Member accounts: receive AWS Config recorder, delivery channel, and custom rules through a service-managed CloudFormation StackSet

## Evidence Location
- This SOP document (primary evidence): `ac-cp-co-main/docs/governance/SECURE_AWS_ACCOUNT_GOVERNANCE_SOP.md`
- Delegated admin + StackSet deployment (supporting evidence):
  - `ac-cp-co-main/docs/getting_started/PREREQUISITES.md`
  - `ac-cp-co-main/docs/getting_started/README.md`
  - `ac-cp-co-main/cdk-infra/lib/main-stack.ts`
  - `ac-cp-co-main/cdk-infra/lib/member-stack.ts`
