resource "aws_dynamodb_table" "repositories_dynamo_table" {
  name           = "${var.environment_name}-repositories-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "OrganizationNodeId"

  attribute {
    name = "OrganizationNodeId"
    type = "S"
  }

  point_in_time_recovery {
    enabled = true
  }

  server_side_encryption {
    enabled = true
  }

  tags = merge(
    local.common_tags,
    {
      "Name"         = "${var.environment_name}-repositories-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
      "name"         = "${var.environment_name}-repositories-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
      "service_name" = var.service_name
    },
  )
}