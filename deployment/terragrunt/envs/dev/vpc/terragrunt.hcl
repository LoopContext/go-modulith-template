include "root" {
  path = find_in_parent_folders()
}

terraform {
  source = "../../../../opentofu/modules/vpc"
}

inputs = {
  environment = "dev"
  vpc_cidr    = "10.0.0.0/16"
}
