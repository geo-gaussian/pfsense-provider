terraform {
  required_providers {
    pfsense-nexus = {
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

provider "pfsense-nexus" {
  url      = var.nexus_url
  username = var.nexus_username
  password = var.nexus_password
}
