module "cluster" {
  source                           = "github.com/jenkins-x-terraform/terraform-jx-azure?ref=v0.1.0"
  apex_domain_integration_enabled  = var.apex_domain_integration_enabled
  apex_domain_name                 = var.apex_domain_name
  apex_resource_group_name         = var.apex_resource_group_name
  cluster_name                     = var.cluster_name
  cluster_network_model            = var.cluster_network_model
  cluster_node_resource_group_name = var.cluster_node_resource_group_name
  cluster_resource_group_name      = var.cluster_resource_group_name
  cluster_version                  = var.cluster_version
  dns_enabled                      = var.dns_enabled
  dns_resource_group_name          = var.dns_resource_group_name
  dns_prefix                       = var.dns_prefix
  domain_name                      = var.domain_name
  enable_log_analytics             = var.enable_log_analytics
  jx_bot_token                     = var.jx_bot_token
  jx_git_url                       = var.jx_git_url
  jx_bot_username                  = var.jx_bot_username
  key_vault_enabled                = var.key_vault_enabled
  key_vault_name                   = var.key_vault_name
  key_vault_resource_group_name    = var.key_vault_resource_group_name
  key_vault_sku                    = var.key_vault_sku
  location                         = var.location
  logging_retention_days           = var.logging_retention_days
  network_name                     = var.network_name
  network_resource_group_name      = var.network_resource_group_name
  node_count                       = var.node_count
  node_size                        = var.node_size
  storage_resource_group_name      = var.storage_resource_group_name
  subnet_cidr                      = var.subnet_cidr
  subnet_name                      = var.subnet_name
  vnet_cidr                        = var.vnet_cidr
}

output "connect" {
  description = "Connect to cluster"
  value       = module.cluster.connect
}
output "follow_install_logs" {
  description = "Follow Jenkins X install logs"
  value       = "jx admin log"
}
output "docs" {
  description = "Follow Jenkins X 3.x docs for more information"
  value       = "https://jenkins-x.io/v3/"
}
output "kube_config_admin" {
  value     = module.cluster.kube_config_admin
  sensitive = true
}