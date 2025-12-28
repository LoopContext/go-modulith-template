include "root" {
  path = find_in_parent_folders()
}

terraform {
  source = "../../../../opentofu/modules/eks"
}

dependency "vpc" {
  config_path = "../vpc"
}

inputs = {
  environment = "dev"
  vpc_id      = dependency.vpc.outputs.vpc_id
  subnet_ids  = dependency.vpc.outputs.private_subnet_ids
}
