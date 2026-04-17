# AWS Config Cost Optimization Conformance Pack - Architectural Analysis

## Executive Summary

This CloudFormation template deploys an organization-wide cost optimization framework using AWS Config, Lambda, Systems Manager, and CloudFormation StackSets. It implements a delegated administration model to monitor and remediate cost-inefficient resources across an entire AWS Organization from a central audit account.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Core Assumptions](#core-assumptions)
3. [Resource Breakdown](#resource-breakdown)
4. [Workflow & Execution Flow](#workflow--execution-flow)
5. [Dependencies & Requirements](#dependencies--requirements)
6. [Scope & Boundaries](#scope--boundaries)
7. [Configuration Parameters](#configuration-parameters)
8. [Security Model](#security-model)
9. [Edge Cases & Limitations](#edge-cases--limitations)
10. [Cost Implications](#cost-implications)
11. [Operational Considerations](#operational-considerations)

---

## Architecture Overview

### High-Level Architecture

```
Management Account (Root)
    │
    ├── Delegated Admin Account (Audit/Security Account)
    │   ├── Main CloudFormation Stack (main.yaml)
    │   │   ├── GetOrgDetailsFunction (Lambda)
    │   │   ├── ShareDocumentFunction (Lambda)
    │   │   ├── EbsGp3Remediation (SSM Document)
    │   │   └── ConformancePack (Organization-wide)
    │   │
    │   └── StackSet Controller
    │       └── Deploys to all member accounts
    │
    └── Member Accounts (via StackSet)
        ├── CustomConfigFunction (Lambda)
        ├── CustomConfigFunctionRole (IAM)
        ├── AutomationRole (IAM)
        └── Config Rules (3 rules per account)
            ├── CostOpt-Ebs-Gp3
            ├── CostOpt-Ebs-Unattached
            └── CostOpt-S3-WithoutLifecycle
```

### Component Interaction Flow

```
1. Resource Change → AWS Config detects change
2. Config Rule triggered → Invokes Lambda Function
3. Lambda evaluates compliance → Returns COMPLIANT/NON_COMPLIANT
4. Non-compliant resource detected → Config Rule stores evaluation
5. Manual/Automatic Remediation → SSM Automation Document executes
6. Remediation uses AutomationRole → Modifies resource
7. Config re-evaluates → Updates compliance status
8. Results aggregated → Visible in delegated admin account
```

---

## Core Assumptions

### Organizational Structure

1. **AWS Control Tower is deployed** - The solution assumes an AWS Organizations setup managed by Control Tower
2. **Delegated administrator model** - Assumes deployment from an audit/security account, not the management account
3. **Active member accounts** - All accounts in the organization are active and accessible
4. **Single region deployment** - The template deploys to a single region specified at deployment time

### Access & Permissions

1. **Trusted access enabled** - StackSets and Config have trusted access with AWS Organizations
2. **Delegated admin registered** - The audit account is registered as delegated administrator for:
   - `config.amazonaws.com`
   - `config-multiaccountsetup.amazonaws.com`
   - `member.org.stacksets.cloudformation.amazonaws.com`
3. **Cross-account access** - Member accounts can invoke Lambda in delegated admin account
4. **SSM document sharing** - All member accounts can access shared SSM documents

### Technical Assumptions

1. **AWS Config is recording** - AWS Config must already be enabled and recording in all accounts
2. **Python 3.12 runtime available** - Lambda functions use Python 3.12 runtime
3. **ARM64 architecture support** - Lambdas use ARM64 for cost efficiency
4. **IAM roles can be created** - Template has permission to create IAM roles with specific names

### Cost Optimization Logic

1. **gp2 volumes are always suboptimal** - Any gp2 volume should be converted to gp3
2. **Unattached volumes are wasteful** - EBS volumes not attached to instances are non-compliant
3. **Lifecycle policies are mandatory** - All S3 buckets should have lifecycle configurations
4. **Manual remediation by default** - Automatic remediation is disabled by default (Automatic: false)

---

## Resource Breakdown

### 1. GetOrgDetailsFunction (AWS::Serverless::Function)

**Purpose**: Retrieves AWS Organization details including root ID and management account ID.

**Key Attributes**:

- **Runtime**: Python 3.12 on ARM64
- **Handler**: `index.handler`
- **Trigger**: CloudFormation Custom Resource lifecycle events (Create/Update/Delete)
- **Permissions**:
  - `organizations:ListRoots`
  - `organizations:DescribeOrganization`

**Functionality**:

```python
# Returns organization root ID as physical resource ID
# Returns management account ID in response data
# Used by OrgDetails custom resource
```

**Dependencies**: None (first resource to execute)

**Outputs**:

- Physical Resource ID: Organization Root ID
- Data Attributes: `MasterAccountId`

---

### 2. OrgDetails (Custom::OrganizationRootId)

**Purpose**: Custom resource that invokes GetOrgDetailsFunction to retrieve organization metadata.

**Key Attributes**:

- **Type**: Custom Resource
- **Service Token**: GetOrgDetailsFunction ARN
- **Lifecycle**: Synchronous (blocks stack creation until complete)

**Usage**:

- Referenced by `ConformancePack` to exclude management account
- Referenced by `StackSet` to target organization root

**Outputs**:

- `!Ref OrgDetails` → Organization Root ID
- `!GetAtt OrgDetails.MasterAccountId` → Management Account ID

---

### 3. StackSet (AWS::CloudFormation::StackSet)

**Purpose**: Deploys infrastructure to all member accounts in the organization.

**Key Attributes**:

- **Permission Model**: `SERVICE_MANAGED` (uses AWS Organizations integration)
- **Deployment Target**: Entire organization root (all accounts)
- **Auto Deployment**: Enabled (automatically deploys to new accounts)
- **Retain on Account Removal**: False (cleans up when accounts leave)
- **Call As**: `DELEGATED_ADMIN` (runs from audit account, not management)
- **Concurrency**: 100% max concurrent, 100% failure tolerance

**Deployment Strategy**:

```yaml
MaxConcurrentPercentage: 100 # Deploy to all accounts simultaneously
FailureTolerancePercentage: 100 # Continue even if some fail
RegionConcurrencyType: PARALLEL # Not relevant (single region)
```

**Resources Deployed to Each Member Account**:

#### 3.1. AutomationRole (IAM Role)

- **Name**: `CostOpt-Automation-{Region}` (hardcoded for conformance pack reference)
- **Purpose**: Assumed by SSM Automation to execute remediations
- **Trust Policy**: `ssm.amazonaws.com`
- **Permissions**:
  - `ec2:ModifyVolume` on all resources (for gp2 → gp3 conversion)

#### 3.2. CustomConfigFunctionRole (IAM Role)

- **Purpose**: Execution role for the Config rule evaluation Lambda
- **Managed Policies**:
  - `AWSLambdaBasicExecutionRole` (CloudWatch Logs)
  - `AWSConfigRulesExecutionRole` (Config integration)
- **Custom Policies**:
  - `s3:GetLifecycleConfiguration` on all buckets

#### 3.3. CustomConfigFunction (Lambda)

- **Name**: `CostOptimizationConformanceConfigRuleFunction` (hardcoded)
- **Runtime**: Python 3.12 on ARM64
- **Purpose**: Evaluates resources against cost optimization rules
- **Evaluation Logic**: Contains three evaluation functions:
  - `ebs_gp3_evaluate_compliance`: Checks if volume is gp2
  - `ebs_unattached_evaluate_compliance`: Checks if volume has attachments
  - `s3_withoutlifecycle_evaluate_compliance`: Checks if bucket has lifecycle rules

**Lambda Execution Flow**:

```python
1. Receives event from AWS Config
2. Determines message type:
   - ScheduledNotification → Batch evaluate all resources
   - ConfigurationItemChangeNotification → Evaluate single resource
3. Calls appropriate evaluation function based on customFunctionPrefix
4. Returns COMPLIANT, NON_COMPLIANT, or NOT_APPLICABLE
5. Puts evaluation results back to Config
```

**Supported Message Types**:

- `ScheduledNotification`: Periodic evaluation of all resources
- `ConfigurationItemChangeNotification`: Real-time evaluation on resource change
- `OversizedConfigurationItemChangeNotification`: Handles large configuration items

#### 3.4. CustomConfigFunctionPermission (Lambda Permission)

- **Purpose**: Allows AWS Config service to invoke the Lambda function
- **Principal**: `config.amazonaws.com`

---

### 4. ShareDocumentFunction (AWS::Serverless::Function)

**Purpose**: Shares SSM Automation Documents from delegated admin account to all member accounts.

**Key Attributes**:

- **Runtime**: Python 3.12 on ARM64
- **Timeout**: 30 seconds
- **Permissions**:
  - `organizations:ListAccounts` (list all org accounts)
  - `ssm:ModifyDocumentPermission` (share documents)

**Functionality**:

```python
# On Create: Share document with all member accounts (batches of 20)
# On Update: Unshare old document, share new document
# On Delete: Unshare document from all accounts
# Filters out: Management account and inactive accounts
```

**Batching Logic**: Processes 20 accounts per API call (AWS API limit)

---

### 5. ConformancePack (AWS::Config::OrganizationConformancePack)

**Purpose**: Deploys Config rules organization-wide from a central template.

**Key Attributes**:

- **Name**: `Cost-Optimization`
- **Scope**: All organization accounts except management account
- **Deployment Method**: Organization-level conformance pack (deployed by AWS Config)

**Input Parameters**:

- `EbsGp3DocumentArn`: ARN of SSM document for remediation

**Contains Three Config Rules**:

#### 5.1. EbsGp3Rule

- **Config Rule Name**: `CostOpt-Ebs-Gp3`
- **Resource Type**: `AWS::EC2::Volume`
- **Trigger**: Configuration change on EBS volumes
- **Owner**: `CUSTOM_LAMBDA`
- **Source**: `CostOptimizationConformanceConfigRuleFunction`
- **Input Parameters**:
  ```json
  {
    "desiredVolumeType": "gp3",
    "customFunctionPrefix": "ebs_gp3"
  }
  ```
- **Evaluation Logic**: Marks gp2 volumes as NON_COMPLIANT

#### 5.2. EbsGp3Remediation (AWS::Config::RemediationConfiguration)

- **Target**: SSM Document (EbsGp3Remediation)
- **Resource Parameter**: `volumeid` → `RESOURCE_ID`
- **Automatic**: False (requires manual approval)
- **Execution Controls**:
  - Concurrent execution: 10% of resources
  - Error tolerance: 10%
- **Retry Logic**:
  - Max attempts: 10
  - Retry interval: 600 seconds (10 minutes)

#### 5.3. EbsUnattachedRule

- **Config Rule Name**: `CostOpt-Ebs-Unattached`
- **Resource Type**: `AWS::EC2::Volume`
- **Trigger**: Configuration change on EBS volumes
- **Input Parameters**:
  ```json
  {
    "customFunctionPrefix": "ebs_unattached"
  }
  ```
- **Evaluation Logic**: Marks volumes without attachments as NON_COMPLIANT
- **No Remediation**: Detection only

#### 5.4. S3WithoutLifecycleRule

- **Config Rule Name**: `CostOpt-S3-WithoutLifecycle`
- **Resource Type**: `AWS::S3::Bucket`
- **Trigger**: Configuration change on S3 buckets
- **Input Parameters**:
  ```json
  {
    "customFunctionPrefix": "s3_withoutlifecycle"
  }
  ```
- **Evaluation Logic**: Marks buckets without lifecycle configuration as NON_COMPLIANT
- **No Remediation**: Detection only

---

### 6. EbsGp3Remediation (AWS::SSM::Document)

**Purpose**: SSM Automation Document that converts gp2 volumes to gp3.

**Key Attributes**:

- **Document Type**: Automation
- **Schema Version**: 0.3
- **Assume Role**: `arn:{{ global:AWS_PARTITION }}:iam::{{ global:ACCOUNT_ID }}:role/CostOpt-Automation-{{ global:REGION }}`

**Parameters**:

- `volumeid` (String): EBS volume ID to modify

**Execution Steps**:

1. **ModifyVolume**: Calls EC2 `ModifyVolume` API
   - Changes volume type to gp3
   - Preserves all other volume settings (size, IOPS, throughput)

**Global Variables Used**:

- `{{ global:AWS_PARTITION }}`: AWS partition (aws, aws-cn, aws-us-gov)
- `{{ global:ACCOUNT_ID }}`: Target account ID where remediation runs
- `{{ global:REGION }}`: Target region
- `{{ volumeid }}`: Parameter passed from Config remediation

**Cross-Account Execution**:

- Document stored in delegated admin account
- Shared with all member accounts via ShareDocumentFunction
- Assumes role in target account during execution

---

### 7. EbsGp3RemediationShare (Custom::ShareDocument)

**Purpose**: Triggers ShareDocumentFunction to share the SSM document with all org accounts.

**Key Attributes**:

- **Type**: Custom Resource
- **Service Token**: ShareDocumentFunction ARN
- **Document Name**: Reference to EbsGp3Remediation

**Lifecycle**:

- **Create**: Shares document with all active accounts
- **Update**: Re-shares if document name changes
- **Delete**: Unshares document from all accounts

---

## Workflow & Execution Flow

### Deployment Flow

```
1. Stack Creation Initiated in Delegated Admin Account
   ↓
2. GetOrgDetailsFunction Created & Executed
   ↓
3. OrgDetails Custom Resource Returns:
   - Organization Root ID
   - Management Account ID
   ↓
4. ShareDocumentFunction Created
   ↓
5. EbsGp3Remediation SSM Document Created
   ↓
6. EbsGp3RemediationShare Custom Resource Executed
   - Shares SSM document with all member accounts
   ↓
7. StackSet Created & Deployed
   - Deploys CustomConfigFunction to each account
   - Deploys IAM roles to each account
   - Deployment happens in parallel across all accounts
   ↓
8. ConformancePack Deployed (DependsOn: StackSet)
   - Creates Config rules in each account
   - Links rules to Lambda function
   - Configures remediation actions
   ↓
9. Stack Creation Complete
```

### Runtime Execution Flow

#### Scenario A: New EBS gp2 Volume Created

```
1. User creates gp2 EBS volume in member account
   ↓
2. AWS Config detects configuration change
   ↓
3. Config Rule "CostOpt-Ebs-Gp3" triggered
   ↓
4. Config invokes CustomConfigFunction Lambda
   Event: {
     messageType: "ConfigurationItemChangeNotification",
     configurationItem: { volumeType: "gp2", ... }
   }
   ↓
5. Lambda executes ebs_gp3_evaluate_compliance()
   - Checks if volumeType == "gp2"
   - Returns "NON_COMPLIANT"
   ↓
6. Lambda calls Config PutEvaluations API
   ↓
7. Config stores evaluation result
   ↓
8. Administrator views non-compliant resource in Config console
   ↓
9. Administrator triggers manual remediation
   ↓
10. Config Remediation invokes SSM Document
    - Cross-account execution from delegated admin
    - Assumes CostOpt-Automation role in member account
    ↓
11. SSM executes ModifyVolume API call
    - Converts volume from gp2 to gp3
    ↓
12. Volume modification in progress (can take minutes)
    ↓
13. Config detects volume configuration change
    ↓
14. Config re-evaluates volume
    - volumeType now "gp3"
    - Returns "COMPLIANT"
    ↓
15. Compliance status updated in Config dashboard
```

#### Scenario B: Scheduled Evaluation

```
1. Config triggers scheduled evaluation (periodic)
   ↓
2. Config invokes Lambda with ScheduledNotification
   ↓
3. Lambda calls get_configuration_items()
   - Lists all resources of type AWS::EC2::Volume
   - Batch retrieves configurations
   ↓
4. Lambda evaluates each volume
   - Builds evaluation for each resource
   ↓
5. Lambda puts all evaluations to Config in single API call
   ↓
6. Config updates compliance status for all volumes
```

### Cross-Account Communication Flow

```
Member Account                     Delegated Admin Account
     │                                      │
     │  1. Config Rule Trigger              │
     │─────────────────────────────────────→│
     │                                      │
     │  2. Lambda Invocation                │
     │     (cross-account)                  │
     │                                      │
     │  3. Lambda Execution                 │
     │     ← Evaluates resource             │
     │                                      │
     │  4. Return Compliance Result         │
     │←─────────────────────────────────────│
     │                                      │
     │  5. Remediation Triggered            │
     │─────────────────────────────────────→│
     │                                      │
     │  6. SSM Document Shared              │
     │     ← Executes in member account     │
     │     ← Assumes AutomationRole         │
     │                                      │
     │  7. Resource Modified                │
     │     (volume converted)               │
     │                                      │
```

---

## Dependencies & Requirements

### Pre-Deployment Requirements

#### AWS Organization Setup

1. **AWS Organizations enabled** with multiple member accounts
2. **AWS Control Tower deployed** (assumed architecture)
3. **Organization structure**:
   - Management account (root)
   - Delegated administrator account (audit/security)
   - One or more member accounts

#### Service Trust Relationships

1. **AWS Config trusted access**:

   ```bash
   aws organizations enable-aws-service-access \
     --service-principal=config-multiaccountsetup.amazonaws.com
   ```

2. **CloudFormation StackSets trusted access**:
   ```bash
   aws organizations enable-aws-service-access \
     --service-principal=member.org.stacksets.cloudformation.amazonaws.com
   ```

#### Delegated Administrator Registration

1. **Config delegated admin**:

   ```bash
   aws organizations register-delegated-administrator \
     --account-id <audit-account-id> \
     --service-principal config.amazonaws.com

   aws organizations register-delegated-administrator \
     --account-id <audit-account-id> \
     --service-principal config-multiaccountsetup.amazonaws.com
   ```

2. **StackSets delegated admin**:
   ```bash
   aws organizations register-delegated-administrator \
     --account-id <audit-account-id> \
     --service-principal member.org.stacksets.cloudformation.amazonaws.com
   ```

#### AWS Config Prerequisites

1. **Config Recorder enabled** in all member accounts
2. **Config Delivery Channel configured** in all accounts
3. **S3 bucket for Config** (delivery destination)
4. **Appropriate IAM role** for Config service

#### IAM Permissions Required for Deployment

The deploying principal (user/role) must have:

- `cloudformation:*` for stack operations
- `lambda:*` for function creation
- `iam:*` for role creation
- `ssm:*` for document creation
- `config:*` for conformance pack creation
- `organizations:*` for organization queries
- `sts:AssumeRole` for cross-account operations

### Resource Dependencies (Within Template)

```
Dependency Graph:

GetOrgDetailsFunction (no dependencies)
    ↓
OrgDetails (depends on GetOrgDetailsFunction)
    ↓
    ├─→ StackSet (depends on OrgDetails)
    │       ↓
    │   ConformancePack (depends on StackSet, EbsGp3Remediation)
    │
    └─→ ConformancePack (uses OrgDetails.MasterAccountId)

ShareDocumentFunction (no dependencies)
    ↓
EbsGp3Remediation (no dependencies)
    ↓
EbsGp3RemediationShare (depends on EbsGp3Remediation, ShareDocumentFunction)

ConformancePack (depends on StackSet completion)
```

**Critical Dependency**: `ConformancePack` has `DependsOn: StackSet` because:

- Lambda function must exist in member accounts before Config rules are created
- Config rules reference the Lambda function by name
- If Config rules deploy before Lambda, they'll fail to invoke

### Runtime Dependencies

#### For Evaluation (Lambda Execution)

- **AWS Config service** must be recording
- **Lambda function** must have network access to AWS APIs
- **IAM permissions** on CustomConfigFunctionRole must be correct
- **Configuration items** must be available in Config

#### For Remediation (SSM Automation)

- **SSM service** must be available in target region
- **AutomationRole** must exist with correct name format
- **SSM document** must be shared with target account
- **EC2 API** must be accessible for volume modification
- **Target account** must have SSM enabled

---

## Scope & Boundaries

### What's Included

#### Monitored Resources

1. **EBS Volumes** (AWS::EC2::Volume)
   - gp2 vs gp3 volume type compliance
   - Attachment status
2. **S3 Buckets** (AWS::S3::Bucket)
   - Lifecycle configuration presence

#### Organizational Scope

- **All member accounts** in the organization
- **Single region** per deployment (specified at stack creation)
- **Excludes management account** (best practice for security)
- **Automatically includes new accounts** (via auto-deployment)

#### Capabilities

- **Detection**: Identifies non-compliant resources
- **Evaluation**: Real-time and scheduled compliance checks
- **Remediation**: Automated fix for gp2 → gp3 conversion (manual trigger)
- **Aggregation**: Central visibility in delegated admin account
- **Reporting**: Standard Config compliance reporting

### What's NOT Included

#### Out of Scope Resources

- **RDS instances** (no cost optimization rules)
- **Lambda functions** (no evaluation)
- **EC2 instances** (no right-sizing rules)
- **Elastic IPs** (no detection)
- **NAT Gateways** (no optimization)
- **Load Balancers** (no evaluation)
- **Other cost optimization opportunities** beyond the three rules

#### Missing Features

- **Automatic remediation** (disabled by default, requires manual trigger)
- **Cost impact reporting** (no savings calculation)
- **Multi-region deployment** (single region only)
- **Custom rule addition** (requires template modification)
- **Notification system** (no SNS/email alerts)
- **Dashboard** (uses native Config console only)
- **Remediation for S3/EBS-Unattached** (detection only)

#### Organizational Limitations

- **Does not work without Organizations** (not for standalone accounts)
- **Requires Control Tower** (assumed in documentation)
- **Management account excluded** (no evaluation of resources there)
- **No support for nested OUs** with different configurations
- **No account-level exemptions** (beyond management account)

#### Technical Limitations

- **Single Lambda function** per account (all rules share it)
- **Hardcoded IAM role names** (can't customize)
- **Hardcoded Lambda function name** (can't customize)
- **No support for resource tagging** in evaluation logic
- **No support for scheduled remediation** (must trigger manually)

---

## Configuration Parameters

### Template Parameters

**None explicitly defined** - This template has no `Parameters` section.

**Implicit Parameters** (from environment):

- `AWS::StackName`: Used to name the StackSet and documents
- `AWS::Region`: Target deployment region
- `AWS::AccountId`: Delegated administrator account ID
- `AWS::Partition`: AWS partition (aws, aws-cn, aws-us-gov)

### Conformance Pack Input Parameters

Defined in `ConformancePack.TemplateBody`:

| Parameter           | Type   | Purpose                         | Value                                                                       |
| ------------------- | ------ | ------------------------------- | --------------------------------------------------------------------------- |
| `EbsGp3DocumentArn` | String | ARN of SSM remediation document | `arn:${Partition}:ssm:${Region}:${AccountId}:document/${EbsGp3Remediation}` |

### Config Rule Parameters

#### EbsGp3Rule

```json
{
  "desiredVolumeType": "gp3",
  "customFunctionPrefix": "ebs_gp3"
}
```

- **desiredVolumeType**: Target volume type (hardcoded to "gp3")
- **customFunctionPrefix**: Function name to call in Lambda

#### EbsUnattachedRule

```json
{
  "customFunctionPrefix": "ebs_unattached"
}
```

#### S3WithoutLifecycleRule

```json
{
  "customFunctionPrefix": "s3_withoutlifecycle"
}
```

### Remediation Configuration Parameters

#### EbsGp3Remediation

```yaml
Parameters:
  volumeid:
    ResourceValue:
      Value: RESOURCE_ID # Automatically populated by Config
ExecutionControls:
  SsmControls:
    ConcurrentExecutionRatePercentage: 10 # Max 10% concurrent
    ErrorPercentage: 10 # Stop if >10% fail
MaximumAutomaticAttempts: 10 # Retry up to 10 times
RetryAttemptSeconds: 600 # Wait 10 min between retries
Automatic: false # Requires manual trigger
```

### Hardcoded Configuration Values

#### IAM Role Names

- **AutomationRole**: `CostOpt-Automation-${AWS::Region}`
  - **Reason**: Referenced by SSM document assume role
  - **Impact**: Cannot deploy multiple instances in same account/region

#### Lambda Function Names

- **CustomConfigFunction**: `CostOptimizationConformanceConfigRuleFunction`
  - **Reason**: Referenced by Config rules
  - **Impact**: Cannot deploy multiple instances

#### Config Rule Names

- `CostOpt-Ebs-Gp3`
- `CostOpt-Ebs-Unattached`
- `CostOpt-S3-WithoutLifecycle`

#### Conformance Pack Name

- `Cost-Optimization`

### Customization Points

#### Easily Customizable

1. **StackSet deployment targets** - Change `OrganizationalUnitIds`
2. **Remediation execution controls** - Adjust concurrency and error thresholds
3. **Automatic remediation** - Change `Automatic: false` to `true`
4. **Retry logic** - Modify `MaximumAutomaticAttempts` and `RetryAttemptSeconds`

#### Requires Code Changes

1. **Adding new rules** - Must modify Lambda inline code
2. **Changing evaluation logic** - Must update Lambda functions
3. **Multi-region deployment** - Must duplicate template per region
4. **Resource name customization** - Must update all references

---

## Security Model

### IAM Roles & Permissions

#### AutomationRole (Per Member Account)

```yaml
AssumeRolePolicyDocument:
  Effect: Allow
  Principal:
    Service: ssm.amazonaws.com # Only SSM can assume
  Action: sts:AssumeRole

Policies:
  - Effect: Allow
    Action: ec2:ModifyVolume
    Resource: "*" # All EBS volumes in account
```

**Security Analysis**:

- ✅ **Least privilege principal** (only SSM service)
- ⚠️ **Broad resource scope** (all volumes, but action is limited)
- ✅ **Limited actions** (only ModifyVolume, can't delete/create)
- ⚠️ **No condition keys** (no resource tags or volume state checks)

**Potential Risks**:

- Could modify volumes unintentionally if SSM document is compromised
- No protection against modifying critical volumes
- No MFA requirement for remediation

**Mitigation Strategies**:

- Use `Automatic: false` to require manual approval
- Implement resource tagging and condition keys
- Monitor CloudTrail for SSM automation executions

#### CustomConfigFunctionRole (Per Member Account)

```yaml
ManagedPolicies:
  - AWSLambdaBasicExecutionRole # CloudWatch Logs write
  - AWSConfigRulesExecutionRole # Config integration

CustomPolicies:
  - Effect: Allow
    Action: s3:GetLifecycleConfiguration
    Resource: "*" # All S3 buckets
```

**Security Analysis**:

- ✅ **Managed policies** for standard functions
- ✅ **Read-only S3 access** (cannot modify buckets)
- ⚠️ **Broad S3 scope** (all buckets in account)
- ✅ **No write permissions** on AWS resources

**AWS Config Integration**:

- `AWSConfigRulesExecutionRole` provides:
  - `config:PutEvaluations` (write compliance results)
  - `config:Get*` (read configuration items)
  - Read-only access to resources for evaluation

#### GetOrgDetailsFunction (Delegated Admin Account)

```yaml
Policies:
  - Effect: Allow
    Action:
      - organizations:ListRoots
      - organizations:DescribeOrganization
    Resource: "*" # Organization APIs don't support resource-level
```

**Security Analysis**:

- ✅ **Read-only permissions**
- ✅ **Minimal scope** (only 2 actions)
- ℹ️ **Wildcard resource** (required by Organizations API design)
- ✅ **Short-lived execution** (only during stack create/update)

#### ShareDocumentFunction (Delegated Admin Account)

```yaml
Policies:
  - Effect: Allow
    Action: organizations:ListAccounts
    Resource: "*"
  - Effect: Allow
    Action: ssm:ModifyDocumentPermission
    Resource: !Sub arn:${AWS::Partition}:ssm:${AWS::Region}:${AWS::AccountId}:document/${AWS::StackName}-*Remediation-*
```

**Security Analysis**:

- ✅ **Scoped SSM permissions** (only documents matching pattern)
- ✅ **Read-only Organizations access**
- ✅ **Cannot modify arbitrary documents**
- ⚠️ **Can share documents with all accounts** (by design)

### Cross-Account Security Model

#### Config Rule Invocation Flow

```
Member Account                 Delegated Admin Account
     │                                 │
     │ AWS Config Service              │
     │  (uses service-linked role)     │
     │                                 │
     │──── Lambda:Invoke ─────────────→│ CustomConfigFunction
     │                                 │  (executes with)
     │                                 │  (Lambda execution role)
     │                                 │
     │←─── Evaluation Result ──────────│
     │                                 │
```

**Permissions Flow**:

1. AWS Config in member account invokes Lambda in admin account
2. Lambda execution role in admin account evaluates resource
3. Lambda calls `config:PutEvaluations` API
4. Results written back to Config in member account

**Critical Permission**: `CustomConfigFunctionPermission`

```yaml
Action: lambda:InvokeFunction
Principal: config.amazonaws.com # Any Config service
FunctionName: !GetAtt CustomConfigFunction.Arn
```

**Security Consideration**: This allows ANY AWS Config service to invoke the Lambda, not just member accounts in the organization.

**Recommended Enhancement**:

```yaml
Condition:
  StringEquals:
    "aws:PrincipalOrgID": "${OrganizationId}"
```

#### SSM Document Sharing Model

```
Delegated Admin Account        Member Account
     │                              │
     │ SSM Document exists          │
     │ (EbsGp3Remediation)          │
     │                              │
     │──── ModifyDocumentPermission │
     │──── (Share with AccountId) ──→│
     │                              │
     │                              │ Config Remediation
     │                              │ triggers automation
     │                              │
     │                              │ SSM executes document
     │                              │  ↓
     │                              │ AssumeRole
     │                              │ (CostOpt-Automation)
     │                              │  ↓
     │                              │ ModifyVolume API call
```

**Security Boundaries**:

1. **Document in admin account** (centrally managed)
2. **Execution in member account** (using member account role)
3. **No cross-account role assumption** from admin to member
4. **Member account retains control** via AutomationRole permissions

### Compliance & Governance

#### CloudFormation Stack Policy

**None defined** - Stacks have no protection against updates or deletion.

**Recommendation**:

```json
{
  "Statement": [
    {
      "Effect": "Deny",
      "Principal": "*",
      "Action": ["Update:Delete", "Update:Replace"],
      "Resource": "*"
    }
  ]
}
```

#### Resource Tags

**None applied** - No tags for cost allocation, ownership, or compliance tracking.

**Recommendation**:

```yaml
Tags:
  - Key: CostCenter
    Value: Security
  - Key: Compliance
    Value: CostOptimization
  - Key: ManagedBy
    Value: CloudFormation
```

#### cfn-lint & cfn-nag Suppressions

The template includes security tool suppressions:

**cfn-nag suppressions**:

```yaml
rules_to_suppress:
  - id: W89
    reason: VPC not required. # Lambda not in VPC
  - id: W92
    reason: Reserved Concurrency not required. # No concurrency limits
  - id: W11
    reason: Resource level permissions not possible on this API. # Wildcard resources
  - id: W28
    reason: Hardcoded name required for Conformance Pack. # Named resources
  - id: W58
    reason: Permissions provided through managed policy AWSLambdaBasicExecutionRole.
```

**Security Impact**:

- **W89 (VPC)**: Lambdas access AWS APIs only, no VPC needed
- **W92 (Concurrency)**: Risk of Lambda throttling under high load
- **W11 (Wildcards)**: Some AWS APIs don't support resource-level permissions
- **W28 (Named resources)**: Required for cross-account references
- **W58 (CloudWatch)**: Permissions provided correctly via managed policy

---

## Edge Cases & Limitations

### Edge Case 1: Volume Already in Modification

**Scenario**: Remediation triggered while volume is already being modified.

**Behavior**:

```python
# EC2 ModifyVolume API call fails with:
VolumeModificationRateExceeded: "Volume vol-xxx is currently being modified"
```

**Impact**:

- Remediation marked as failed
- Config retry logic kicks in (10 attempts, 10 min intervals)
- Manual intervention may be needed if persistent

**Mitigation**:

- SSM automation doesn't check volume state before modification
- No pre-flight validation
- Consider adding check step in SSM document

### Edge Case 2: Volume Attached to Stopped Instance

**Scenario**: EBS volume attached to stopped EC2 instance.

**Behavior**:

- Rule evaluation: Volume has attachments → COMPLIANT (for Unattached rule)
- Remediation: ModifyVolume succeeds (can modify while instance stopped)

**Consideration**: Volume type change doesn't require instance to be running.

### Edge Case 3: Encrypted Volume with Customer-Managed Key

**Scenario**: gp2 volume encrypted with CMK, conversion to gp3 attempted.

**Behavior**:

- ModifyVolume preserves encryption settings
- Requires AutomationRole to have `kms:CreateGrant` on CMK
- **Current template DOES NOT grant KMS permissions**

**Impact**: Remediation fails with KMS access denied.

**Fix Required**:

```yaml
AutomationRole:
  Policies:
    - PolicyName: KmsAccess
      PolicyDocument:
        Statement:
          - Effect: Allow
            Action:
              - kms:CreateGrant
              - kms:Decrypt
              - kms:DescribeKey
            Resource: "*"
            Condition:
              StringEquals:
                "kms:ViaService": "ec2.${AWS::Region}.amazonaws.com"
```

### Edge Case 4: Volume Size Below gp3 Minimum

**Scenario**: gp2 volume smaller than 1 GiB (theoretical minimum).

**Behavior**:

- gp3 minimum size: 1 GiB
- gp2 minimum size: 1 GiB
- No issue in practice

**Note**: Both volume types have same size constraints.

### Edge Case 5: High-Performance gp2 Volume (>3,000 IOPS)

**Scenario**: Large gp2 volume (>1TB) with burst IOPS >3,000.

**Consideration**:

- gp2: 3 IOPS/GB, max 16,000 IOPS
- gp3: 3,000 IOPS baseline (included), configurable up to 16,000
- Conversion to gp3 with default settings may reduce performance

**Remediation Behavior**:

```yaml
ModifyVolume:
  VolumeType: gp3
  # IOPS not specified → defaults to 3,000
  # Throughput not specified → defaults to 125 MB/s
```

**Impact**: Large volumes may see IOPS reduction unless explicitly configured.

**Fix Required**: Add IOPS calculation logic

```yaml
# Calculate equivalent IOPS
Iops: "{{ calculateIops(volumeSize) }}" # Not currently supported
```

### Edge Case 6: Member Account Leaves Organization

**Scenario**: Account removed from organization after stack deployment.

**Behavior**:

- `RetainStacksOnAccountRemoval: false` → StackSet instance deleted
- Resources in leaving account are deleted:
  - Lambda function removed
  - IAM roles removed
  - Config rules remain but become non-functional

**Impact**:

- Config rules can't evaluate (Lambda doesn't exist)
- Remediation can't execute (IAM role deleted)
- Account loses cost optimization monitoring

**Consideration**: Leaving accounts should manually delete conformance pack resources.

### Edge Case 7: New Account Joins Organization

**Scenario**: New account added to organization.

**Behavior**:

- `AutoDeployment.Enabled: true` → Automatic stack instance creation
- Resources deployed to new account within minutes
- Conformance pack automatically includes new account
- Config rules become active immediately

**Requirement**: New account must have AWS Config already enabled.

### Edge Case 8: Lambda Function Quota Exceeded

**Scenario**: Delegated admin account at Lambda function limit.

**Behavior**:

- StackSet deployment fails in admin account
- Member account stacks can't reference Lambda (doesn't exist)
- Config rules fail to evaluate

**Mitigation**:

- Lambda function limit: 1,000 per region per account
- This template only creates 1 Lambda in admin account
- Unlikely unless account has many other functions

### Edge Case 9: Config Rule Evaluation During Remediation

**Scenario**: Volume modification triggered, Config evaluates before completion.

**Timeline**:

```
T+0:  Remediation starts → ModifyVolume API call
T+1:  Volume state: "optimizing"
T+2:  Config detects configuration change
T+3:  Config evaluates volume
      → volumeType: "gp3"
      → state: "optimizing"
      → Evaluation: COMPLIANT
T+60: Volume modification complete
```

**Behavior**: Config marks volume as compliant immediately, even while optimizing.

**Note**: This is correct behavior - volume type change is recorded immediately in Config.

### Edge Case 10: S3 Bucket with Versioning and Lifecycle

**Scenario**: S3 bucket has lifecycle rules for current versions but not non-current versions.

**Evaluation Logic**:

```python
if "Rules" not in lifecycle_configuration or len(lifecycle_configuration["Rules"]) == 0:
    return "NON_COMPLIANT"
return "COMPLIANT"
```

**Behavior**: Any lifecycle rule causes COMPLIANT status.

**Limitation**: Doesn't validate quality of lifecycle rules (e.g., rules that don't save costs).

### Edge Case 11: Region Not Supported by gp3

**Scenario**: Deployment in region where gp3 is not available.

**Current Regions**: gp3 available in all commercial AWS regions (as of 2025).

**Behavior**: If deployed in unsupported region:

- Rule evaluation works
- Remediation fails (ModifyVolume API error)

**Note**: Not a practical concern in 2025.

### Edge Case 12: Lambda Function Code Size

**Scenario**: Custom Config function inline code approaching 4,096 character limit.

**Current Size**: ~350 lines of Python (within limit).

**Risk**: Adding more rules could exceed inline code limit.

**Mitigation**:

- Current: `Code.ZipFile` (inline)
- If exceeded: Use `Code.S3Bucket` and `Code.S3Key`

### Edge Case 13: Concurrent Remediation Limit

**Scenario**: 1,000 gp2 volumes marked non-compliant, all remediated simultaneously.

**Configuration**:

```yaml
ConcurrentExecutionRatePercentage: 10 # Max 100 concurrent
ErrorPercentage: 10 # Stop if >10% fail
```

**Behavior**:

- Max 100 volumes modified concurrently
- If >10 volumes fail → Entire remediation batch stops
- Remaining volumes not processed

**Consideration**: SSM Automation execution quota: 100 concurrent executions per account.

### Edge Case 14: StackSet Deployment Failure in One Account

**Scenario**: StackSet instance fails to deploy in one member account.

**Configuration**:

```yaml
FailureTolerancePercentage: 100 # Continue even if all fail
```

**Behavior**:

- Deployment continues to other accounts
- Failed account skipped
- ConformancePack still deploys (doesn't check StackSet instance status)

**Impact**:

- Config rules created in failed account
- Lambda function doesn't exist
- All evaluations fail

**Mitigation**: Monitor StackSet operation status, manually retry failed instances.

### Edge Case 15: Organization Root ID Changes

**Scenario**: Organization structure reorganized, root ID changes (rare).

**Behavior**:

- `OrgDetails` custom resource returns old root ID
- StackSet targets old root (may fail)

**Mitigation**: Update stack to refresh OrgDetails.

### Edge Case 16: SSM Document Sharing Limit

**Scenario**: ShareDocumentFunction processes accounts in batches of 20.

**API Limit**: `ModifyDocumentPermission` limited to 20 accounts per call.

**Behavior**:

```python
for batch in range(0, len(accounts), 20):
    ssm.modify_document_permission(
        AccountIdsToAdd=accounts[batch : batch + 20]
    )
```

**Edge Case**: If organization has >20 accounts, multiple API calls required.

**Risk**: If one batch fails, subsequent batches not processed.

**Current Handling**: No error handling for partial failures.

---

## Cost Implications

### Deployment Costs

#### Per Delegated Admin Account

| Resource                       | Quantity | Monthly Cost (us-east-1) | Notes                                |
| ------------------------------ | -------- | ------------------------ | ------------------------------------ |
| Lambda (GetOrgDetailsFunction) | 1        | $0.00                    | Only executes during stack lifecycle |
| Lambda (ShareDocumentFunction) | 1        | $0.00                    | Only executes during stack lifecycle |
| SSM Document                   | 1        | $0.00                    | Free to store                        |
| CloudFormation Stack           | 1        | $0.00                    | Free                                 |
| Organization Conformance Pack  | 1        | $0.00                    | Free                                 |

**Subtotal: $0.00/month**

#### Per Member Account (via StackSet)

| Resource                      | Quantity | Monthly Cost      | Notes                           |
| ----------------------------- | -------- | ----------------- | ------------------------------- |
| Lambda (CustomConfigFunction) | 1        | ~$0.20            | ARM64, invoked per Config event |
| Lambda invocations            | Variable | $0.20/1M          | Depends on resource change rate |
| IAM Roles                     | 2        | $0.00             | Free                            |
| Config Rules                  | 3        | $6.00             | $2/rule/region/account          |
| Config Evaluations            | Variable | $0.003/evaluation | Real-time + scheduled           |
| SSM Automation Executions     | Variable | $0.002/automation | Only during remediation         |

**Estimated per account: $6-10/month** (depending on evaluation frequency)

**Total for 100 accounts: $600-1,000/month**

### Config Cost Breakdown

**AWS Config Pricing** (us-east-1):

- **Config Rule**: $2.00/rule/month
- **Config Evaluation**: $0.003/evaluation (after first 100,000 free)
- **Conformance Pack**: Included (no additional charge beyond rules)

**Example Calculation** (per account):

```
3 rules × $2.00 = $6.00/month
1,000 evaluations/day × 30 days = 30,000 evaluations
30,000 evaluations × $0.003 = $0.09/month (under free tier)

Total: $6.09/month per account
```

### Lambda Cost Breakdown

**Lambda Pricing** (ARM64, us-east-1):

- **Invocation**: $0.20 per 1M requests
- **Duration**: $0.0000133334 per GB-second
- **ARM64 discount**: ~20% cheaper than x86_64

**CustomConfigFunction Sizing**:

- Memory: 128 MB (default)
- Average duration: <500ms
- Invocations: Depends on resource change rate

**Example Calculation**:

```
Scenario: 1,000 EBS volumes, 10% change daily
- Evaluations: 100/day × 30 days = 3,000/month
- Cost: 3,000 / 1,000,000 × $0.20 = $0.0006
- Duration: 3,000 × 0.5s × 0.125GB × $0.0000133334 = $0.0025

Total: $0.003/month (negligible)
```

### SSM Automation Cost

**SSM Automation Pricing**:

- **Standard automation**: $0.002 per automation step
- **Cross-account automation**: Same pricing

**EbsGp3Remediation Steps**: 1 step

**Example Calculation**:

```
Scenario: 100 gp2 volumes remediated/month
- Remediations: 100
- Cost: 100 × 1 step × $0.002 = $0.20/month
```

### EBS Cost Savings (Benefit)

**gp2 vs gp3 Pricing** (us-east-1):

- **gp2**: $0.10/GB-month
- **gp3**: $0.08/GB-month
- **Savings**: 20%

**Example Savings Calculation**:

```
Scenario: 10TB of gp2 volumes converted to gp3
- Before: 10,000 GB × $0.10 = $1,000/month
- After: 10,000 GB × $0.08 = $800/month
- Savings: $200/month = $2,400/year

ROI: $200 savings - $6 Config cost = $194/month net savings
Payback period: Immediate
```

### Cost Optimization Potential

#### Rule 1: EBS gp2 → gp3

- **Savings**: 20% on volume storage
- **Additional IOPS**: 3,000 baseline (vs 3 IOPS/GB)
- **Additional throughput**: 125 MB/s baseline

#### Rule 2: EBS Unattached

- **Savings**: 100% of volume cost if deleted
- **Detection only** (no auto-delete for safety)

#### Rule 3: S3 Without Lifecycle

- **Savings**: Varies widely (depends on lifecycle policy)
- **Detection only** (no auto-remediation)

### Total Cost Ownership (TCO)

**For a 100-account organization**:

**Monthly Costs**:

- Config rules: 100 accounts × $6 = $600
- Lambda executions: ~$20
- SSM automations: ~$5
- **Total: $625/month = $7,500/year**

**Potential Savings**:

- 10TB gp2 → gp3: $2,400/year
- 5TB unattached volumes deleted: $6,000/year
- S3 lifecycle optimization: Variable (potentially $10,000+/year)
- **Potential Total: $18,000+/year**

**Net Benefit: $10,500+/year**

**ROI: 140%+**

---

## Operational Considerations

### Deployment

#### Pre-Deployment Checklist

- [ ] AWS Organizations enabled
- [ ] Control Tower deployed (optional but assumed)
- [ ] Delegated admin account identified
- [ ] Trusted access enabled for Config and StackSets
- [ ] Delegated administrator registered
- [ ] AWS Config enabled in all accounts
- [ ] IAM permissions for deploying principal
- [ ] Region selected (single region deployment)

#### Deployment Time

- **Stack creation**: 5-10 minutes
- **StackSet propagation**: 10-30 minutes (depending on account count)
- **Conformance pack deployment**: 5-15 minutes per account
- **Total**: 20-60 minutes for complete deployment

#### Deployment Command

```bash
aws cloudformation deploy \
  --template-file main.yaml \
  --stack-name CostOptimizationConfPack \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM \
  --region us-east-1
```

### Monitoring

#### CloudWatch Metrics to Monitor

1. **Lambda Errors**:

   - `CustomConfigFunction` errors
   - `GetOrgDetailsFunction` errors
   - `ShareDocumentFunction` errors

2. **Config Metrics**:

   - Non-compliant resource count
   - Evaluation failures
   - Remediation success/failure rate

3. **SSM Automation Metrics**:
   - Automation execution status
   - Automation failures

#### Logging

**Lambda Logs** (CloudWatch Logs):

- `/aws/lambda/CostOptimizationConformanceConfigRuleFunction`
- Retention: Not specified (defaults to never expire)
- **Recommendation**: Set retention to 30 days

**CloudFormation Logs**:

- StackSet operation logs
- Custom resource execution logs

**Config Logs**:

- Config evaluation results
- Remediation execution logs
- Available in Config console

**SSM Logs**:

- Automation execution history
- Available in Systems Manager console

#### Alerting Recommendations

**Critical Alerts**:

1. Lambda function errors > 5% of invocations
2. StackSet deployment failures
3. Conformance pack deployment failures
4. Custom resource failures (OrgDetails, ShareDocument)

**Warning Alerts**:

1. High non-compliant resource count
2. Remediation failure rate > 10%
3. Config evaluation delays

### Maintenance

#### Regular Maintenance Tasks

**Monthly**:

- Review non-compliant resources
- Analyze remediation success rate
- Check for failed StackSet instances

**Quarterly**:

- Review and optimize Config rule logic
- Update Lambda runtime if deprecated
- Review IAM role permissions

**Annually**:

- Review cost savings achieved
- Evaluate new cost optimization rules to add
- Assess multi-region expansion

#### Updates & Upgrades

**Adding New Rules**:

1. Modify `CustomConfigFunction` inline code
2. Add new evaluation function
3. Update `ConformancePack` template body
4. Add new Config rule definition
5. Update stack (triggers StackSet update)

**Modifying Existing Rules**:

1. Update Lambda evaluation logic
2. Update Config rule parameters
3. Stack update triggers Lambda redeployment
4. Config rules automatically use new Lambda code

**Changing Remediation Logic**:

1. Update SSM document content
2. Unshare old document
3. Share new document
4. Update conformance pack parameter

### Troubleshooting

#### Issue: Config Rule Evaluation Failures

**Symptoms**:

- Config rule shows "Evaluation failed"
- Resources not marked compliant/non-compliant

**Diagnosis**:

```bash
# Check Lambda logs
aws logs tail /aws/lambda/CostOptimizationConformanceConfigRuleFunction --follow

# Check Config rule status
aws configservice describe-config-rules \
  --config-rule-names CostOpt-Ebs-Gp3

# Test Lambda manually
aws lambda invoke \
  --function-name CostOptimizationConformanceConfigRuleFunction \
  --payload file://test-event.json \
  response.json
```

**Common Causes**:

- Lambda function doesn't exist (StackSet deployment failed)
- Lambda IAM role missing permissions
- Lambda timeout (increase if needed)
- Bug in evaluation logic

#### Issue: Remediation Doesn't Execute

**Symptoms**:

- Remediation triggered but nothing happens
- SSM automation shows "Failed" status

**Diagnosis**:

```bash
# Check SSM automation execution
aws ssm describe-automation-executions \
  --filters Key=DocumentName,Values=<DocumentName>

# Check automation execution details
aws ssm get-automation-execution \
  --automation-execution-id <execution-id>
```

**Common Causes**:

- AutomationRole doesn't exist
- AutomationRole missing permissions
- SSM document not shared with account
- Resource already being modified
- Encrypted volume without KMS permissions

#### Issue: StackSet Instance Deployment Failed

**Symptoms**:

- StackSet shows "OUTDATED" status
- Some accounts missing Lambda function

**Diagnosis**:

```bash
# List StackSet instances
aws cloudformation list-stack-instances \
  --stack-set-name CostOptimizationConfPack-StackSet

# Get instance details
aws cloudformation describe-stack-instance \
  --stack-set-name CostOptimizationConfPack-StackSet \
  --stack-instance-account <account-id> \
  --stack-instance-region <region>
```

**Common Causes**:

- Account at CloudFormation stack limit
- IAM role creation failed (name conflict)
- Lambda creation failed (quota exceeded)
- StackSet execution role missing permissions

**Resolution**:

```bash
# Retry failed instances
aws cloudformation update-stack-instances \
  --stack-set-name CostOptimizationConfPack-StackSet \
  --accounts <failed-account-id> \
  --regions <region>
```

#### Issue: New Accounts Not Automatically Included

**Symptoms**:

- New account added to organization
- No Config rules deployed

**Diagnosis**:

```bash
# Check StackSet auto-deployment
aws cloudformation describe-stack-set \
  --stack-set-name CostOptimizationConfPack-StackSet \
  --query 'StackSet.AutoDeployment'
```

**Common Causes**:

- AutoDeployment not enabled
- Account not in target OU
- AWS Config not enabled in new account

**Resolution**:

1. Verify AutoDeployment.Enabled: true
2. Check account is under organization root
3. Enable AWS Config in new account
4. Wait 5-10 minutes for automatic deployment

### Disaster Recovery

#### Backup Strategy

**What to Backup**:

- CloudFormation template (version controlled)
- Custom Lambda code (in template)
- SSM document definition (in template)
- Config rule parameters (in template)

**No Data to Backup**: All configuration is in the template.

#### Recovery Scenarios

**Scenario 1: Accidental Stack Deletion**

**Impact**:

- All resources deleted
- Config rules stop working
- Remediation documents deleted
- Member account stacks may remain (orphaned)

**Recovery Steps**:

```bash
# Re-deploy main stack
aws cloudformation deploy \
  --template-file main.yaml \
  --stack-name CostOptimizationConfPack \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM

# StackSet will re-sync to member accounts
# Conformance pack will re-deploy
```

**Recovery Time**: 30-60 minutes

**Scenario 2: StackSet Corruption**

**Impact**:

- StackSet can't update member accounts
- Resources in member accounts intact but stale

**Recovery Steps**:

```bash
# Delete StackSet instances
aws cloudformation delete-stack-instances \
  --stack-set-name CostOptimizationConfPack-StackSet \
  --accounts <account-ids> \
  --regions <region> \
  --no-retain-stacks

# Update stack to recreate StackSet
aws cloudformation update-stack \
  --stack-name CostOptimizationConfPack \
  --template-body file://main.yaml \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM
```

**Scenario 3: Conformance Pack Drift**

**Impact**:

- Config rules don't match template
- Evaluation logic inconsistent

**Recovery Steps**:

```bash
# Delete conformance pack
aws configservice delete-organization-conformance-pack \
  --organization-conformance-pack-name Cost-Optimization

# Update stack to recreate
aws cloudformation update-stack \
  --stack-name CostOptimizationConfPack \
  --template-body file://main.yaml \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM
```

### Scaling Considerations

#### Account Scaling

- **Current**: Supports all accounts in organization
- **Limit**: Organizations support up to 5,000 accounts
- **Bottleneck**: SSM document sharing (20 accounts per API call)
- **Solution**: ShareDocumentFunction already batches in groups of 20

#### Resource Scaling

- **EBS Volumes**: No limit on evaluation count
- **S3 Buckets**: No limit on evaluation count
- **Scheduled Evaluations**: May take longer with more resources
- **Remediation**: Capped at 10% concurrency by default

#### Regional Scaling

- **Current**: Single region deployment
- **Multi-region Strategy**:
  1. Deploy separate stack per region
  2. Use unique stack names (include region)
  3. Conformance pack per region
  4. IAM roles unique per region (already includes `${AWS::Region}`)

---

## Conclusion

This CloudFormation template implements a sophisticated, organization-wide cost optimization framework with the following characteristics:

**Strengths**:

- ✅ Centralized management from delegated admin account
- ✅ Automatic deployment to new accounts
- ✅ Serverless architecture (low operational overhead)
- ✅ ARM64 Lambda for cost efficiency
- ✅ Manual remediation approval for safety
- ✅ Comprehensive Config integration

**Weaknesses**:

- ⚠️ Single region deployment only
- ⚠️ Limited to three cost optimization rules
- ⚠️ No KMS permissions for encrypted volumes
- ⚠️ Hardcoded resource names (limits flexibility)
- ⚠️ No automatic remediation by default
- ⚠️ No notification system
- ⚠️ Broad Lambda invoke permission (not org-scoped)

**Best Use Cases**:

- Organizations using AWS Control Tower
- Cost-conscious teams wanting automated monitoring
- Security teams managing delegated administration
- Teams with >10 accounts needing centralized compliance

**Not Recommended For**:

- Single AWS accounts (over-engineered)
- Multi-region deployments (requires multiple stacks)
- Organizations without AWS Config enabled
- Teams requiring automatic remediation (needs modification)

**Estimated Value**:

- **Implementation effort**: 1-2 days
- **Monthly cost**: $6-10 per account
- **Potential savings**: $200+ per 10TB of EBS
- **ROI**: 140%+ with moderate EBS usage

This solution provides a solid foundation for organization-wide cost optimization monitoring and remediation, with room for customization and expansion based on specific organizational needs.
