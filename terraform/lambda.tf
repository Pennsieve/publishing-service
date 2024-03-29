resource "aws_lambda_function" "service_lambda" {
  description       = "Lambda Function which handles requests for the serverless Publishing Service"
  function_name     = "${var.environment_name}-${var.service_name}-service-lambda-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
  handler           = "publishing_service"
  runtime           = "go1.x"
  role              = aws_iam_role.publishing_service_lambda_role.arn
  timeout           = 300
  memory_size       = 128
  s3_bucket         = var.lambda_bucket
  s3_key            = "${var.service_name}/${var.service_name}-${var.image_tag}.zip"

  vpc_config {
    subnet_ids         = tolist(data.terraform_remote_state.vpc.outputs.private_subnet_ids)
    security_group_ids = [data.terraform_remote_state.platform_infrastructure.outputs.upload_v2_security_group_id]
  }

  environment {
    variables = {
      ENV = var.environment_name
      PENNSIEVE_DOMAIN = data.terraform_remote_state.account.outputs.domain_name
      REGION = var.aws_region
      PUBLISHING_INFO_TABLE = aws_dynamodb_table.publishing_info_dynamo_table.name
      REPOSITORIES_TABLE = aws_dynamodb_table.repositories_dynamo_table.name
      REPOSITORY_QUESTIONS_TABLE = aws_dynamodb_table.repository_questions_dynamo_table.name
      DATASET_PROPOSAL_TABLE = aws_dynamodb_table.dataset_proposals_dynamo_table.name
      RDS_PROXY_ENDPOINT        = data.terraform_remote_state.pennsieve_postgres.outputs.rds_proxy_endpoint
      EMAIL_TEMPLATE_BUCKET  = data.terraform_remote_state.platform_infrastructure.outputs.dataset_assets_bucket_id
      EMAIL_TEMPLATE_SUBMITTED = "PublishingService/EmailTemplates/dataset-proposal-submitted.html"
      EMAIL_TEMPLATE_WITHDRAWN = "PublishingService/EmailTemplates/dataset-proposal-withdrawn.html"
      EMAIL_TEMPLATE_ACCEPTED = "PublishingService/EmailTemplates/dataset-proposal-accepted.html"
      EMAIL_TEMPLATE_REJECTED = "PublishingService/EmailTemplates/dataset-proposal-rejected.html"
    }
  }
}
