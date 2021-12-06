module "eks-jx" {
  source          = "github.com/jenkins-x/terraform-aws-eks-jx?ref=v1.15.12"
  cluster_name    = var.cluster_name
  cluster_version = var.cluster_version
  region          = var.region
  vault_user      = var.vault_user
  is_jx2          = false
  jx_git_url      = var.jx_git_url
  jx_bot_username = var.jx_bot_username
  jx_bot_token    = var.jx_bot_token
  force_destroy   = var.force_destroy
}
