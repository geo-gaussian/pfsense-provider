# 🧭 pfsense_dns_resolver (Data Source)

Reads the DNS Resolver configuration, including host and domain overrides and
access controls. Live appliance compatibility remains unverified.

## 🧪 Example Usage

```terraform
data "pfsense_dns_resolver" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive API response. Use `jsondecode()` to access fields.

Custom options may contain secrets. Protect Terraform state and saved plans.
