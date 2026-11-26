terraform {
  required_providers {
    pfsense = { source = "geo-gaussian/pfsense-nexus" }
  }
}

variable "nexus_url" { type = string }
variable "nexus_username" { type = string }
variable "nexus_password" {
  type      = string
  sensitive = true
}
variable "nexus_insecure" { type = bool }
variable "alias_name" { type = string }
variable "alias_description" { type = string }

provider "pfsense" {
  url      = var.nexus_url
  username = var.nexus_username
  password = var.nexus_password
  insecure = var.nexus_insecure
}

resource "pfsense_firewall_alias" "test" {
  name    = var.alias_name
  type    = "host"
  descr   = var.alias_description
  targets = [{ name = "example.test" }]
}
