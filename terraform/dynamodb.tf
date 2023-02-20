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

resource "aws_dynamodb_table" "repository_questions_dynamo_table" {
  name           = "${var.environment_name}-repository-questions-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "Id"

  attribute {
    name = "Id"
    type = "N"
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
      "Name"         = "${var.environment_name}-repository-questions-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
      "name"         = "${var.environment_name}-repository-questions-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
      "service_name" = var.service_name
    },
  )
}

resource "aws_dynamodb_table" "dataset_proposals_dynamo_table" {
  name           = "${var.environment_name}-dataset-proposals-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "UserId"
  range_key      = "ProposalId"

  attribute {
    name = "UserId"
    type = "N"
  }

  attribute {
    name = "ProposalId"
    type = "N"
  }

  attribute {
    name = "RepositoryId"
    type = "N"
  }

  global_secondary_index {
    name               = "RepositoryIdIndex"
    hash_key           = "RepositoryId"
    range_key          = "UserId"
    projection_type    = "ALL"
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
      "Name"         = "${var.environment_name}-dataset-proposals-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
      "name"         = "${var.environment_name}-dataset-proposals-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
      "service_name" = var.service_name
    },
  )

}