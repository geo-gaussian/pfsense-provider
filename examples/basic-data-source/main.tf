terraform {
  required_providers {
    pfsense = {
      source = "geo-gaussian/pfsense-nexus"
    }
  }
}

variable "nexus_url" { type = string }
variable "nexus_username" { type = string }
variable "nexus_password" {
  type      = string
  sensitive = true
}

provider "pfsense" {
  url      = var.nexus_url
  username = var.nexus_username
  password = var.nexus_password
}

data "pfsense_firewall_aliases" "current" {}

data "pfsense_interfaces" "current" {}

data "pfsense_firewall_nat_port_forward" "current" {}

output "firewall_aliases" {
  value = data.pfsense_firewall_aliases.current.aliases
}

output "interfaces" {
  value = jsondecode(data.pfsense_interfaces.current.json)
}

output "port_forward_rules" {
  value = jsondecode(data.pfsense_firewall_nat_port_forward.current.json)
}
