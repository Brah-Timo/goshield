terraform {
  required_version = ">= 1.8"
  required_providers {
    azurerm = { source = "hashicorp/azurerm", version = "~> 3.110" }
    helm    = { source = "hashicorp/helm",    version = "~> 2.14"  }
    kubernetes = { source = "hashicorp/kubernetes", version = "~> 2.31" }
  }
  backend "azurerm" {
    resource_group_name  = "goshield-tfstate"
    storage_account_name = "goshieldtfstate"
    container_name       = "tfstate"
    key                  = "goshield.terraform.tfstate"
  }
}

provider "azurerm" {
  features {}
  subscription_id = var.subscription_id
}

# ── Resource Group ─────────────────────────────────────────────────────────
resource "azurerm_resource_group" "goshield" {
  name     = var.resource_group_name
  location = var.location
  tags     = local.tags
}

# ── AKS Cluster ───────────────────────────────────────────────────────────
resource "azurerm_kubernetes_cluster" "aks" {
  name                = "goshield-aks-${var.environment}"
  location            = azurerm_resource_group.goshield.location
  resource_group_name = azurerm_resource_group.goshield.name
  dns_prefix          = "goshield-${var.environment}"
  kubernetes_version  = "1.29"
  sku_tier            = var.environment == "prod" ? "Standard" : "Free"

  default_node_pool {
    name                = "system"
    node_count          = var.system_node_count
    vm_size             = var.system_vm_size
    os_disk_size_gb     = 50
    type                = "VirtualMachineScaleSets"
    enable_auto_scaling = true
    min_count           = 1
    max_count           = var.system_node_count * 2
    zones               = ["1", "2", "3"]
  }

  identity {
    type = "SystemAssigned"
  }

  network_profile {
    network_plugin    = "azure"
    network_policy    = "azure"
    load_balancer_sku = "standard"
  }

  monitor_metrics {}
  tags = local.tags
}

# ── Workload node pool (for AI service) ───────────────────────────────────
resource "azurerm_kubernetes_cluster_node_pool" "workload" {
  name                  = "workload"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.aks.id
  vm_size               = var.workload_vm_size
  node_count            = 1
  enable_auto_scaling   = true
  min_count             = 1
  max_count             = 4
  zones                 = ["1", "2", "3"]
  node_labels           = { "workload" = "ai" }
  tags                  = local.tags
}

# ── Azure Container Registry ───────────────────────────────────────────────
resource "azurerm_container_registry" "acr" {
  name                = "goshield${var.environment}acr"
  resource_group_name = azurerm_resource_group.goshield.name
  location            = azurerm_resource_group.goshield.location
  sku                 = "Standard"
  admin_enabled       = false
  tags                = local.tags
}

resource "azurerm_role_assignment" "aks_acr_pull" {
  scope                = azurerm_container_registry.acr.id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_kubernetes_cluster.aks.kubelet_identity[0].object_id
}

# ── Azure Database for PostgreSQL Flexible Server ─────────────────────────
resource "azurerm_postgresql_flexible_server" "pg" {
  name                   = "goshield-pg-${var.environment}"
  resource_group_name    = azurerm_resource_group.goshield.name
  location               = azurerm_resource_group.goshield.location
  version                = "16"
  administrator_login    = "goshield"
  administrator_password = var.pg_password
  storage_mb             = 32768
  sku_name               = var.environment == "prod" ? "GP_Standard_D4s_v3" : "B_Standard_B1ms"
  backup_retention_days  = 7
  geo_redundant_backup_enabled = var.environment == "prod"
  zone                   = "1"
  tags                   = local.tags
}

resource "azurerm_postgresql_flexible_server_database" "goshield" {
  name      = "goshield"
  server_id = azurerm_postgresql_flexible_server.pg.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}

# ── Azure Cache for Redis ─────────────────────────────────────────────────
resource "azurerm_redis_cache" "redis" {
  name                = "goshield-redis-${var.environment}"
  location            = azurerm_resource_group.goshield.location
  resource_group_name = azurerm_resource_group.goshield.name
  capacity            = var.environment == "prod" ? 2 : 0
  family              = var.environment == "prod" ? "C" : "C"
  sku_name            = var.environment == "prod" ? "Standard" : "Basic"
  enable_non_ssl_port = false
  minimum_tls_version = "1.2"
  tags                = local.tags
}

locals {
  tags = {
    project     = "goshield"
    environment = var.environment
    managed_by  = "terraform"
  }
}
