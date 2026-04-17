# Data Encryption and Key Management SOP (NETSEC-002)

This SOP defines encryption-in-transit, encryption-at-rest, and key management practices for customer engagements delivered with the **AWS Config Cost Optimization Conformance Pack Project**.

## Purpose
Standardize how we:
- Encrypt traffic for any internet-exposed endpoints
- Encrypt outbound requests to external endpoints
- Enforce encryption at rest for AWS services that store data
- Manage cryptographic keys using a dedicated key management solution

## Scope
Applies to customer projects and to the governance solution components deployed by this project.

## 1) Endpoints Exposed to the Internet (Inbound) — Encryption in Transit

**Policy:** Any endpoint exposed to the Internet must enforce TLS.

Implementation guidance:
- Use TLS 1.2+ (or customer-required minimum) and disable weak ciphers.
- Use AWS Certificate Manager (ACM) for certificate lifecycle management.
- Terminate TLS only at approved entry points (CloudFront, ALB/NLB TLS listeners, API Gateway custom domains).
- Enforce HTTPS-only (redirect HTTP to HTTPS where applicable).

Evidence to capture:
- Endpoint inventory (DNS name → service → TLS policy → certificate source)
- Configuration evidence (IaC snippet or screenshots) for TLS listeners/policies

**Project baseline:** The **AWS Config Cost Optimization Conformance Pack Project** does not deploy customer-facing internet endpoints by default (no public API Gateway/ALB endpoints).

## 2) Requests to External Endpoints (Outbound) — Encryption in Transit

**Policy:** All outbound requests to external endpoints must use HTTPS/TLS with certificate validation.

Implementation guidance:
- Use HTTPS with certificate validation (no TLS verification bypass).
- Prefer AWS PrivateLink/VPC endpoints for AWS service access when customers require private routing.
- If outbound internet egress is required, use customer-approved egress controls (NAT + firewall/proxy where required) and log egress.

Evidence to capture:
- External dependency inventory (endpoint → purpose → encryption method)
- Egress control decision and logging destination

**Project baseline:** Solution runtime communications are to AWS service APIs using AWS SDK/CLI over TLS; no third-party outbound dependencies are required.

## 3) Encryption at Rest (Service-Native Encryption)

**Policy:** Enable native encryption at rest for any AWS service that stores data unless there is a documented exception.

Implementation guidance:
- Prefer service-native defaults (S3 SSE, EBS/RDS/DynamoDB encryption, Secrets Manager encryption, etc.).
- Document any exception with business reason, compensating controls, and approval.

**Project example implementation:** AWS Config delivery is written to an S3 bucket configured with:
- Encryption at rest: S3-managed encryption (SSE-S3)
- In-transit enforcement: TLS-only access (`enforceSSL: true`)

## 4) Key Management (Dedicated Key Management Solution)

**Policy:** Cryptographic keys must be stored and managed using a dedicated key management solution.

Implementation guidance:
- Default to **AWS Key Management Service (AWS KMS)** customer managed keys (CMKs) for regulated/sensitive data.
- Enable rotation where required and enforce least privilege via IAM/key policies.
- AWS-managed service keys are acceptable only when customer policy and data classification permit (for example, S3 SSE-S3).

**Project key management note:** The project’s default S3 delivery bucket uses AWS-managed keys (SSE-S3). Where customer policy requires CMKs, the engagement uses AWS KMS CMKs (SSE-KMS) per this SOP.

## Evidence Location
- This SOP (primary evidence): `ac-cp-co-main/docs/governance/DATA_ENCRYPTION_AND_KEY_MANAGEMENT_SOP_NETSEC-002.md`
- S3 delivery bucket encryption + TLS-only enforcement: `ac-cp-co-main/cdk-infra/lib/member-stack.ts`
- Solution architecture context: `ac-cp-co-main/template/ARCHITECTURE_ANALYSIS.md`