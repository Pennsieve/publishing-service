
// Create log group for timeseries-service API Lambda.
resource "aws_cloudwatch_log_group" "publishing_service_api_lambda_log_group" {
  name              = "/aws/lambda/${aws_lambda_function.service_lambda.function_name}"
  retention_in_days = 30
  tags              = local.common_tags
}

# PUBLISHING SERVICE API GATEWAY LOG GROUP
resource "aws_cloudwatch_log_group" "publishing-service-gateway-log-group" {
  name              = "${var.environment_name}/${var.service_name}/publishing-api-gateway"
  retention_in_days = 30
}
