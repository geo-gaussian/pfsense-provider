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
variable "alias_description" {
  type    = string
  default = null
}

provider "pfsense" {
  url      = var.nexus_url
  username = var.nexus_username
  password = var.nexus_password
  insecure = true
}

resource "pfsense_firewall_alias" "test" {
  name    = "example"
  type    = "host"
  descr   = var.alias_description
  targets = [{ name = "example.test" }]
}
