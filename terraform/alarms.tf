# T1 CloudWatch alarms (EPIC 868m2zvjt; standard sets from
# pennsieve-infra-dashboard/docs/alarm-coverage-plan.md). No alarm_actions
# yet — dashboard/console-visible only.
module "service_alarms" {
  source = "git@github.com:Pennsieve/terraform-modules.git//service-alarms"

  environment_name = var.environment_name
  service_name     = var.service_name

  lambdas = {
    service = {
      function_name   = aws_lambda_function.service_lambda.function_name
      timeout_seconds = aws_lambda_function.service_lambda.timeout
    }
  }

  dynamodb_tables = {
    publishing-info      = aws_dynamodb_table.publishing_info_dynamo_table.name
    repositories         = aws_dynamodb_table.repositories_dynamo_table.name
    repository-questions = aws_dynamodb_table.repository_questions_dynamo_table.name
    dataset-proposals    = aws_dynamodb_table.dataset_proposals_dynamo_table.name
  }
}
