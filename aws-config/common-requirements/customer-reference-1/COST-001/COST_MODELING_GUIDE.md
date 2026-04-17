# Cost Modeling Guide (TCO + Business Value)

## Purpose
Provide a repeatable method to build a Total Cost of Ownership (TCO) estimate and business value analysis for customer engagements.

This guide is used for engagements delivered with the **AWS Config Cost Optimization Conformance Pack Project**.

## What to Produce (Minimum Artifacts)
1. **Inputs**: deployment scale and usage assumptions (right sizing) + pricing sources (right pricing)
2. **Cost Model**: monthly/annual operating cost estimate before implementation
3. **Business Value**: quantified savings potential and ROI / payback narrative

## Step 1 — Define Inputs (Right Sizing)
Capture the variables that drive cost:
- Number of AWS accounts in scope
- Number of AWS regions in scope
- Estimated resource counts (e.g., EBS volumes, S3 buckets)
- AWS Config evaluation frequency (and expected change rate)
- Expected remediation volume (how many resources remediated per month)
- Data retention assumptions (for AWS Config delivery S3 buckets)

## Step 2 — Define Pricing Assumptions (Right Pricing)
Use region-specific AWS public pricing for:
- AWS Config rule evaluations and recorder costs
- AWS Lambda invocations and duration
- AWS Systems Manager Automation step costs
- Amazon S3 storage class prices for Standard, Standard-IA, Glacier
- The customer’s relevant resource unit prices for savings analysis (e.g., EBS gp2 vs gp3)

Document:
- Region used (e.g., us-east-1)
- Price points used and the date captured

## Step 3 — Build the Operating Cost Model (Before Implementation)
Compute:
- **Config cost**: estimated per account/per region
- **Lambda cost**: expected evaluations × duration × memory
- **SSM Automation cost**: remediation executions × steps × per-step price
- **S3 cost**: storage volume × storage class pricing (include lifecycle transitions if used)

Output:
- Monthly total
- Annual total
- Key cost drivers and sensitivities (what changes cost the most)

## Step 4 — Build Business Value / Value Stream Mapping
Quantify benefits in the customer’s context:
- **Direct savings**: reduced unit costs (e.g., EBS gp2 → gp3 price/GB-month)
- **Cost avoidance**: deleting unattached/orphaned resources (e.g., unattached EBS)
- **Storage optimization**: lifecycle policies that reduce long-term storage cost

Output:
- Savings per unit (per TB, per bucket, etc.)
- Organization-wide savings estimate (scale the unit savings by customer volume)
- ROI and payback narrative

## Step 5 — Validate and Iterate After Deployment
- Compare estimated vs actual (billing + operational metrics)
- Update assumptions (resource counts, evaluation frequency, remediation rates)
- Report deltas and adjust the value narrative

## Where This Is Evidenced in the Project (Example)
The **AWS Config Cost Optimization Conformance Pack Project** includes pre-built cost model examples and business value calculations:
- `template/ARCHITECTURE_ANALYSIS.md` (TCO, savings, ROI examples)
- `docs/architecture_decision_record/0006-enable-aws-config-recorder.md` (per-account/per-region cost ranges)
- `docs/architecture_decision_record/0008-s3-lifecycle-policies.md` (storage cost model + savings)
- `docs/architecture_decision_record/README.md` (cost impact roll-up summary)
