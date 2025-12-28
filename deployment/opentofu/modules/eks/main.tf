resource "aws_iam_role" "cluster" {
  name = "${var.project_name}-eks-cluster-role-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = { Service = "eks.amazonaws.com" }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "cluster_AmazonEKSClusterPolicy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
  role       = aws_iam_role.cluster.name
}

resource "aws_eks_cluster" "main" {
  name     = "${var.project_name}-cluster-${var.environment}"
  role_arn = aws_iam_role.cluster.arn

  vpc_config {
    subnet_ids = var.subnet_ids
  }

  depends_on = [aws_iam_role_policy_attachment:cluster_AmazonEKSClusterPolicy]
}

resource "aws_iam_role" "nodes" {
  name = "${var.project_name}-eks-node-role-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "nodes_AmazonEKSWorkerNodePolicy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  role       = aws_iam_role.nodes.name
}

resource "aws_iam_role_policy_attachment" "nodes_AmazonEKS_CNI_Policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
  role       = aws_iam_role.nodes.name
}

resource "aws_iam_role_policy_attachment" "nodes_AmazonEC2ContainerRegistryReadOnly" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
  role       = aws_iam_role.nodes.name
}

resource "aws_eks_node_group" "main" {
  cluster_name    = aws_eks_cluster.main.name
  node_group_name = "${var.project_name}-nodes-${var.environment}"
  node_role_arn   = aws_iam_role.nodes.arn
  subnet_ids      = var.subnet_ids

  scaling_config {
    desired_size = var.desired_size
    max_size     = var.desired_size + 2
    min_size     = 1
  }

  instance_types = var.instance_types

  depends_on = [
    aws_iam_role_policy_attachment:nodes_AmazonEKSWorkerNodePolicy,
    aws_iam_role_policy_attachment:nodes_AmazonEKS_CNI_Policy,
    aws_iam_role_policy_attachment:nodes_AmazonEC2ContainerRegistryReadOnly,
  ]
}

output "cluster_endpoint" {
  value = aws_eks_cluster.main.endpoint
}

output "cluster_name" {
  value = aws_eks_cluster.main.name
}
