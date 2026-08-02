output "aks_cluster_name"        { value = azurerm_kubernetes_cluster.aks.name }
output "aks_kube_config_raw"     { value = azurerm_kubernetes_cluster.aks.kube_config_raw; sensitive = true }
output "acr_login_server"        { value = azurerm_container_registry.acr.login_server }
output "pg_fqdn"                 { value = azurerm_postgresql_flexible_server.pg.fqdn }
output "redis_hostname"          { value = azurerm_redis_cache.redis.hostname }
output "redis_primary_access_key" { value = azurerm_redis_cache.redis.primary_access_key; sensitive = true }
