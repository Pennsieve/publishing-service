resource "aws_apigatewayv2_api" "publishing-service-api" {
  name          = "Open Repository API"
  protocol_type = "HTTP"
  description = "API for the lambda-based open-repositories API"
  cors_configuration {
    allow_origins = ["*"]
    allow_methods = ["*"]
    allow_headers = ["*"]
    expose_headers = ["*"]
    max_age = 300
  }
  body          = templatefile("${path.module}/publishing_service.yml", {
    authorize_lambda_invoke_uri = data.terraform_remote_state.api_gateway.outputs.authorizer_lambda_invoke_uri
    gateway_authorizer_role = data.terraform_remote_state.api_gateway.outputs.authorizer_invocation_role
    publishing_service_lambda_arn = aws_lambda_function.service_lambda.arn
  })
}

resource "aws_apigatewayv2_api_mapping" "publishing-service-api-map" {
  api_id          = aws_apigatewayv2_api.publishing-service-api.id
  domain_name     = var.api_domain_name
  stage           = aws_apigatewayv2_stage.repository-service-gateway-stage.id
  api_mapping_key = "publishing"

}

resource "aws_apigatewayv2_stage" "repository-service-gateway-stage" {
  api_id = aws_apigatewayv2_api.publishing-service-api.id

  name        = "$default"
  auto_deploy = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.publishing-service-gateway-log-group.arn

    format = jsonencode({
      requestId               = "$context.requestId"
      sourceIp                = "$context.identity.sourceIp"
      requestTime             = "$context.requestTime"
      protocol                = "$context.protocol"
      httpMethod              = "$context.httpMethod"
      resourcePath            = "$context.resourcePath"
      routeKey                = "$context.routeKey"
      status                  = "$context.status"
      responseLength          = "$context.responseLength"
      integrationErrorMessage = "$context.integrationErrorMessage"
    }
    )
  }
}

resource "aws_apigatewayv2_integration" "int" {
  api_id           = aws_apigatewayv2_api.publishing-service-api.id
  integration_type = "AWS_PROXY"
  connection_type = "INTERNET"
  integration_method = "POST"
  integration_uri = aws_lambda_function.service_lambda.invoke_arn
}

resource "aws_lambda_permission" "repository-service-lambda-permission" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.service_lambda.function_name
  principal     = "apigateway.amazonaws.com"

  source_arn = "${aws_apigatewayv2_api.publishing-service-api.execution_arn}/*/*"
}