# 🧭 pfsense_coredns (Data Source)

Reads the CoreDNS configuration. Live appliance compatibility remains unverified.

## 🧪 Example Usage

```terraform
data "pfsense_coredns" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive API response. Use `jsondecode()` to access fields.

Protect Terraform state and saved plans.
