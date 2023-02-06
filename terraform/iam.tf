resource "aws_iam_role" "publishing_service_lambda_role" {
  name = "${var.environment_name}-${var.service_name}-publishing-service-lambda-role-${data.terraform_remote_state.region.outputs.aws_region_shortname}"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "publishing_service_lambda_iam_policy_attachment" {
  role       = aws_iam_role.publishing_service_lambda_role.name
  policy_arn = aws_iam_policy.publishing_service_lambda_iam_policy.arn
}

resource "aws_iam_policy" "publishing_service_lambda_iam_policy" {
  name   = "${var.environment_name}-${var.service_name}-model-service-lambda-iam-policy-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
  path   = "/"
  policy = data.aws_iam_policy_document.publishing_service_iam_policy_document.json
}

data "aws_iam_policy_document" "publishing_service_iam_policy_document" {

  statement {
    sid    = "PublishingServiceLambda-Logs-Permissions"
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutDestination",
      "logs:PutLogEvents",
      "logs:DescribeLogStreams"
    ]
    resources = ["*"]
  }

}