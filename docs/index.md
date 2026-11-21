# pfSense Nexus Provider

The experimental pfSense Nexus provider manages firewall aliases and reads
alias metadata through the Nexus controller's local firewall API.

!> Firewall alias writes and reloads have only been tested against mock servers.
Do not use the resource on a production firewall.

## 🧪 Example Usage

```terraform
terraform {
  required_providers {
    pfsense = {
      source  = "geo-gaussian/pfsense-nexus"
      version = "0.1.0"
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
