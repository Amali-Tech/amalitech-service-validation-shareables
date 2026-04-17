# Network Security SOP (NETSEC-001)

This SOP defines network security best practices for customer environments delivered with the **AWS Config Cost Optimization Conformance Pack Project**.

## Purpose
Standardize how we secure traffic in and within Amazon VPCs and how we apply additional AWS security services for network protection.

## Scope
Applies to:
- Customer VPC design and day-2 changes (subnets, routing, ingress/egress)
- Security groups (internet-facing and east/west)
- Network ACLs (stateless subnet controls)
- Network security services (visibility, threat detection, and policy enforcement)

## 1) Security Groups — Internet ↔ VPC Traffic

**Policy:** Deny by default; allow only required inbound from known sources.

Implementation guidance:
- Do not use `0.0.0.0/0` or `::/0` inbound unless explicitly required and approved.
- Prefer ingress from:
  - Specific CIDR ranges (corporate IPs, known partner networks)
  - Load balancers (SG-to-SG referencing)
  - AWS-managed prefix lists and VPC endpoints where applicable
- Restrict management access:
  - No direct SSH/RDP from the internet; use Session Manager or bastions with strict controls.
- Egress controls:
  - Restrict outbound where possible (e.g., only required ports/destinations).

Verification evidence (examples):
- A security group review record showing intended inbound/outbound rules
- Exception record where public ingress is required

## 2) Security Groups — Intra-VPC (East/West) Traffic

**Policy:** Segment by tier; only allow necessary east/west flows.

Implementation guidance:
- Use separate security groups per tier (web/app/data/shared services).
- Allow east/west using SG referencing rather than broad CIDRs.
- Use least privilege ports/protocols; remove unused rules.

Verification evidence (examples):
- Architecture diagram showing tier segmentation
- Security group matrix (source SG → destination SG → ports)

## 3) Network ACLs — Subnet Inbound/Outbound Restrictions

**Policy:** Use NACLs as a coarse, stateless subnet boundary; security groups remain the primary control.

Implementation guidance:
- Apply baseline NACLs per subnet type:
  - Public subnets: tightly restrict inbound (e.g., only from load balancer subnets or approved CIDRs)
  - Private subnets: restrict inbound to known internal CIDRs/subnets and required ports
- Explicitly document ephemeral port requirements for return traffic.
- Use change control for NACL updates (high blast radius).

Verification evidence (examples):
- NACL rule set documented per subnet type
- Change log/ticket referencing NACL modifications

## 4) Other AWS Security Services for Network Security

Recommended services and use cases:
- **VPC Flow Logs**: network visibility and incident investigation
- **AWS Network Firewall** (if required): centralized inspection and egress filtering
- **AWS Firewall Manager**: policy enforcement at scale (SG baselines, WAF/Shield/Network Firewall policies)
- **Amazon GuardDuty**: threat detection for suspicious activity
- **AWS WAF / AWS Shield** (internet-facing apps): L7 protection and DDoS resilience
- **AWS PrivateLink / VPC Endpoints**: reduce internet exposure for service access

Verification evidence (examples):
- Flow logs enabled and delivered to customer log destination
- GuardDuty enabled organization-wide
- Firewall Manager policy assignments (if used)

## How This SOP Is Applied in This Project

The **AWS Config Cost Optimization Conformance Pack Project** itself is deployed as a serverless governance solution (AWS Config, Lambda, SSM Automation, StackSets) and does not require provisioning or changing customer VPC networking.

During customer engagements:
- We validate that VPC network security is implemented following this SOP for the customer workloads in scope.
- We document any required exceptions (e.g., public ingress) and the compensating controls.

## Evidence Location
- This SOP (primary evidence): `ac-cp-co-main/docs/governance/NETWORK_SECURITY_SOP_NETSEC-001.md`
- Project baseline assumptions (Control Tower): `ac-cp-co-main/docs/getting_started/PREREQUISITES.md`
- Project architecture notes (serverless/no VPC requirement): `ac-cp-co-main/template/ARCHITECTURE_ANALYSIS.md`
