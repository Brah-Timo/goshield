variable "subscription_id"     { type = string }
variable "resource_group_name" { type = string; default = "goshield-rg" }
variable "location"            { type = string; default = "eastus2" }
variable "environment"         { type = string; default = "dev"; validation { condition = contains(["dev","staging","prod"], var.environment); error_message = "Must be dev, staging, or prod." } }
variable "system_node_count"   { type = number; default = 3 }
variable "system_vm_size"      { type = string; default = "Standard_D2s_v3" }
variable "workload_vm_size"    { type = string; default = "Standard_D4s_v3" }
variable "pg_password"         { type = string; sensitive = true }
