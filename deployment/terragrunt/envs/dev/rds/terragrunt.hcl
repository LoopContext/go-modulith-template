include "root" {
  path = find_in_parent_folders()
}

terraform {
  source = "../../../../opentofu/modules/rds"
}

dependency "vpc" {
  config_path = "../vpc"
}

inputs = {
  environment        = "dev"
  vpc_id             = dependency.vpc.outputs.vpc_id
  private_subnet_ids = dependency.vpc.outputs.private_subnet_ids
  db_password        = "change-me-in-production"
}
