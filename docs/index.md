# 🌐 pfSense Nexus Provider

The experimental pfSense Nexus provider manages firewall aliases and reads
selected firewall, network, service, VPN, and system configuration or status
through the Nexus controller's local firewall API. Read-only data sources
return JSON for use with Terraform's `jsondecode()` function. Their individual
pages are listed under [`data-sources/`](data-sources/).

!> Firewall alias writes and reloads have only been tested against mock servers.
Do not use the resource on a production firewall.

## 🧪 Example Usage

```terraform
terraform {
  required_providers {
    pfsense = {
      source  = "geo-gaussian/pfsense-nexus"
      version = ">= 0.1.1"
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
```

## 🔍 Read-only Data Sources

The read-only catalog includes firewall NAT and rule configuration, interface
and route discovery, service settings and status, VPN status, DHCP leases,
system health, and monitoring information. These data sources do not apply
configuration or reload the firewall. Some outputs are marked sensitive, and
WireGuard status omits key material. The additional data sources have only
been tested against mock servers; appliance behaviour and compatibility remain
unverified.

## ⚙️ Argument Reference

- `url` (Required) HTTPS origin of the Nexus controller.
- `username` (Required) Nexus username.
- `password` (Required, Sensitive) Nexus password. Supply it securely.
- `insecure` (Optional) Disables TLS certificate verification. Use only in an
  isolated test environment; verification is enabled by default.

The provider assumes the local firewall is registered as `localhost`. Login and
the alias list have been tested against one real appliance; compatibility with
other installations is not established. Terraform state and saved plans may
contain sensitive information and must be protected.
