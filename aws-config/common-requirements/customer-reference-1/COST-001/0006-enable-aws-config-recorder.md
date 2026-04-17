# ADR-0005: Enable AWS Config Recorder and Delivery Channel in Member Accounts

## Status

Accepted

## Context

After successfully deploying Lambda functions and AWS Config rules to member accounts, validation revealed that while the Lambda function existed, the Config rules had no triggers and were not evaluating resources. Navigation to AWS Config in a member account showed "Set up AWS Config" - indicating Config service was not enabled.

AWS Config requires three components to function:

1. Configuration Recorder - Records resource configurations
2. Delivery Channel - Delivers configuration snapshots to S3
3. Config Rules - Evaluates resource compliance

Initial implementation only created Config rules (component 3), assuming AWS Config was already enabled in member accounts. This assumption was incorrect.

## Decision Drivers

- Config rules require active Config service
- Cannot assume Config is pre-enabled in member accounts
- Config generates significant historical data requiring storage
- Multi-region deployment requires Config in each region
- Cost optimization for Config data storage needed
- Compliance and audit trail requirements

## Considered Options

### Option 1: Assume Config is Pre-Enabled

**Pros:**

- Simpler stack template
- No additional S3 buckets
- Existing Config infrastructure reused

**Cons:**

- DOES NOT WORK - Config not enabled in most accounts
- Rules deploy but never execute
- Silent failure - hard to diagnose
- Inconsistent across accounts
- No guarantee of proper Config setup

### Option 2: Document Manual Config Setup

**Pros:**

- No code changes needed
- Flexibility in Config configuration

**Cons:**

- Manual work for 100+ accounts
- Error-prone process
- Inconsistent configurations
- High operational overhead
- Does not scale

### Option 3: Enable Config Recorder and Delivery Channel in Stack

**Pros:**

- Guaranteed Config enablement
- Consistent configuration across accounts
- Automated setup - no manual intervention
- S3 bucket lifecycle policies for cost optimization
- Works immediately after deployment
- Audit trail from day one

**Cons:**

- Additional S3 buckets in each account/region combination
- Increased stack complexity
- Storage costs for Config data
- Stack takes longer to deploy

## Decision

Enable AWS Config Recorder and Delivery Channel as part of the member account stack deployment.

## Rationale

1. **Functional Requirement**: Config rules cannot work without Config service enabled
2. **Automation**: Eliminates manual setup across 100+ accounts
3. **Consistency**: Ensures identical Config setup in all accounts
4. **Cost Visibility**: Explicit cost management through lifecycle policies
5. **Audit Compliance**: Configuration history available from deployment date
6. **Regional Support**: Each region gets its own Config recorder (AWS requirement)

## Implementation

### S3 Bucket for Config Delivery

```typescript
const configBucket = new s3.Bucket(this, "ConfigBucket", {
  bucketName: `aws-config-bucket-${cdk.Aws.ACCOUNT_ID}-${cdk.Aws.REGION}`,
  removalPolicy: cdk.RemovalPolicy.RETAIN, // Preserve compliance data
  blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
  encryption: s3.BucketEncryption.S3_MANAGED,
  enforceSSL: true, // Security requirement
  bucketKeyEnabled: true, // Cost optimization
  lifecycleRules: [
    {
      enabled: true,
      transitions: [
        {
          storageClass: s3.StorageClass.INFREQUENT_ACCESS,
          transitionAfter: cdk.Duration.days(90), // Optimize costs
        },
        {
          storageClass: s3.StorageClass.GLACIER,
          transitionAfter: cdk.Duration.days(365), // Long-term archive
        },
      ],
    },
  ],
});
```

### Config Recorder

```typescript
const configRecorder = new config.CfnConfigurationRecorder(
  this,
  "ConfigRecorder",
  {
    roleArn: configRole.roleArn,
    recordingGroup: {
      allSupported: true, // Record all resource types
      includeGlobalResourceTypes: true, // Include IAM, CloudFront, etc.
    },
  },
);
```

### Delivery Channel

```typescript
const deliveryChannel = new config.CfnDeliveryChannel(
  this,
  "ConfigDeliveryChannel",
  {
    s3BucketName: configBucket.bucketName,
  },
);

// Ensure proper dependency order
deliveryChannel.addDependency(configRecorder);
```

### Config Rules Dependencies

```typescript
// Ensure rules are created after Config is fully operational
ebsGp3Rule.node.addDependency(configRecorder);
ebsGp3Rule.node.addDependency(deliveryChannel);
// ... repeat for other rules
```

## Consequences

### Positive

- Config rules now execute correctly
- Automatic configuration recording in all accounts
- Compliance history from deployment date
- Cost-optimized storage with lifecycle policies
- No manual Config setup required
- Consistent Config configuration across organization

### Negative

- S3 storage costs for Config data (mitigated by lifecycle policies)
- Additional stack resources increase deployment time
- S3 buckets created in each account/region (300+ buckets for 100 accounts × 3 regions)
- Stack deletion leaves Config data (intentional - RETAIN policy)

### Neutral

- Config data retention managed by lifecycle rules
- Regional Config recorders required (AWS limitation)
- One S3 bucket per account per region

## Cost Analysis

### S3 Storage Costs (per account, per region)

- First 30 days: Standard storage (~$0.023/GB)
- Days 31-90: Standard storage
- Days 91-365: Infrequent Access (~$0.0125/GB)
- After 365 days: Glacier (~$0.004/GB)

### Estimated Monthly Costs (per account per region)

- Config service: $0.003 per configuration item recorded
- S3 storage: $0.10-$2.00 depending on resource count
- Total: Approximately $0.50-$3.00 per account per region

### Organization Total (100 accounts × 3 regions)

- Approximate monthly cost: $150-$900
- Cost justified by compliance and automated resource tracking

## Security Considerations

### S3 Bucket Security

- **Block Public Access**: All public access blocked
- **SSL Enforcement**: enforceSSL: true (denies unencrypted access)
- **Encryption**: S3-managed encryption (SSE-S3)
- **Bucket Key**: Enabled for reduced KMS costs
- **Bucket Policy**: Restrictive policy for Config service only

### IAM Permissions

- Config role uses AWS managed policy: service-role/ConfigRole
- Minimal permissions principle
- Service principal: config.amazonaws.com

## Operational Considerations

### Regional Behavior

- AWS Config is regional service
- Each region requires separate recorder and delivery channel
- Stack deployed to us-east-1, eu-west-1, eu-central-1
- Total: 3 Config recorders per account

### Data Retention

- S3 bucket uses RETAIN removal policy
- Config history preserved even if stack deleted
- Manual cleanup required if full deletion needed

### Monitoring

- Config delivery notifications available via SNS (not implemented)
- CloudWatch metrics for Config service
- S3 bucket metrics for storage usage

## Validation

- Deployed to member account <test-account-id> 
- Config console no longer shows "Set up AWS Config"
- Config rules visible in console with "Compliant/Non-compliant" status
- Lambda function has Config service trigger
- S3 bucket receiving Config snapshots
- Configuration items being recorded

## Related Decisions

- [ADR-0003: Use Inline Lambda Code](0003-use-inline-lambda-code.md)
- [ADR-0005: Use SERVICE_MANAGED StackSets](0005-use-service-managed-stacksets.md)
- [ADR-0008: Implement S3 Lifecycle Policies](0008-s3-lifecycle-policies.md)

## References

- AWS Config Recorder: https://docs.aws.amazon.com/config/latest/developerguide/stop-start-recorder.html
- Config Delivery Channel: https://docs.aws.amazon.com/config/latest/developerguide/manage-delivery-channel.html
- Config Pricing: https://aws.amazon.com/config/pricing/
- S3 Lifecycle Policies: https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-lifecycle-mgmt.html
