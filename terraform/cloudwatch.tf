
// Create log group for timeseries-service API Lambda.
resource "aws_cloudwatch_log_group" "publishing_service_api_lambda_log_group" {
  name              = "/aws/lambda/${aws_lambda_function.service_lambda.function_name}"
  retention_in_days = 30
  tags = local.common_tags
}

// Send logs from publishing-service lambda to Datadog
resource "aws_cloudwatch_log_subscription_filter" "publishing_service_lambda_datadog_subscription" {
  name            = "${aws_cloudwatch_log_group.publishing_service_api_lambda_log_group.name}-subscription"
  log_group_name  = aws_cloudwatch_log_group.publishing_service_api_lambda_log_group.name
  filter_pattern  = ""
  destination_arn = data.terraform_remote_state.region.outputs.datadog_delivery_stream_arn
  role_arn        = data.terraform_remote_state.region.outputs.cw_logs_to_datadog_logs_firehose_role_arn
}

# PUBLISHING SERVICE API GATEWAY LOG GROUP
resource "aws_cloudwatch_log_group" "publishing-service-gateway-log-group" {
  name =  "${var.environment_name}/${var.service_name}/publishing-api-gateway"
  retention_in_days = 30
}