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
