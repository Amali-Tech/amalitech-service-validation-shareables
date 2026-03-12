module "iam" {
  source                                  = "./modules/iam"
  project_name                            = var.project_name
  tags                                    = var.tags
  snapshot_cross_region_copy_function_arn = module.lambda.snapshot_cross_region_copy_function.arn
  snapshot_cleaner_function_arn           = module.lambda.snapshot_cleaner_function.arn
}

module "lambda" {
  source                                = "./modules/lambda"
  project_name                          = var.project_name
  tags                                  = var.tags
  primary_region                        = var.primary_region
  dr_region                             = var.dr_region
  cleaner_lambda_role_arn               = module.iam.cleaner_lambda_role_arn
  copy_lambda_role_arn                  = module.iam.copy_lambda_role_arn
  creator_lambda_role_arn               = module.iam.creator_lambda_role_arn
  restore_rds_lambda_role_arn           = module.iam.restore_rds_lambda_role_arn
  route53_update_record_lambda_role_arn = module.iam.route53_update_record_lambda_role_arn
  rds_instance_id                       = data.aws_db_instance.primary_db.id
  dr_security_group_id                  = module.vpc.dr_security_group_id
  dr_subnet_group_name                  = module.vpc.dr_subnet_group_name
  dr_parameter_group_name               = module.vpc.dr_parameter_group_name
  route53_hosted_zone_id                = module.route53.route53_hosted_zone_id
  route53_database_record_ttl           = module.route53.route53_database_record.ttl
  route53_database_record_name          = module.route53.route53_database_record.name
  providers = {
    aws.dr = aws.dr
  }
}

module "eventbridge" {
  source                                   = "./modules/eventbridge"
  project_name                             = var.project_name
  tags                                     = var.tags
  snapshot_creator_function_arn            = module.lambda.snapshot_creator_function.arn
  snapshot_creator_function_name           = module.lambda.snapshot_creator_function.function_name
  snapshot_cross_region_copy_function_arn  = module.lambda.snapshot_cross_region_copy_function.arn
  snapshot_cross_region_copy_function_name = module.lambda.snapshot_cross_region_copy_function.function_name
  snapshot_cleaner_function_arn            = module.lambda.snapshot_cleaner_function.arn
  snapshot_cleaner_function_name           = module.lambda.snapshot_cleaner_function.function_name
  route53_update_record_function_arn       = module.lambda.route53_update_record_function.arn
  route53_update_record_function_name      = module.lambda.route53_update_record_function.function_name
  rds_instance_id                          = data.aws_db_instance.primary_db.id
  rds_failure_alarm_name                   = module.cloudwatch.rds_failure_alarm_name
  sns_rds_restore_topic_arn                = module.sns.sns_rds_restore_topic_arn

  providers = {
    aws.dr = aws.dr
  }
}

module "cloudwatch" {
  source                  = "./modules/cloudwatch"
  project_name            = var.project_name
  tags                    = var.tags
  rds_instance_id         = data.aws_db_instance.primary_db.id
  sns_dr_topic_arn        = module.sns.sns_dr_topic_arn
  route53_health_check_id = module.route53.route53_health_check_id
  providers = {
    aws.useast1 = aws.useast1
  }
}

module "vpc" {
  source                = "./modules/vpc"
  project_name          = var.project_name
  tags                  = var.tags
  dr_availability_zones = data.aws_availability_zones.dr_availability_zones.names

  providers = {
    aws.dr = aws.dr
  }
}

module "sns" {
  source                     = "./modules/sns"
  project_name               = var.project_name
  tags                       = var.tags
  restore_rds_function_arn   = module.lambda.restore_rds_function.arn
  restore_rds_function_name  = module.lambda.restore_rds_function.function_name
  notification_email_address = var.sns_email

  providers = {
    aws.dr      = aws.dr
    aws.useast1 = aws.useast1
  }
}

module "route53" {
  source               = "./modules/route53"
  project_name         = var.project_name
  tags                 = var.tags
  primary_region       = var.primary_region
  dr_region            = var.dr_region
  primary_vpc_id       = data.aws_subnet.db_subnet.vpc_id
  dr_vpc_id            = module.vpc.dr_vpc_id
  rds_instance_address = data.aws_db_instance.primary_db.address

  providers = {
    aws.dr = aws.dr
  }
}
