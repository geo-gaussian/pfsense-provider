# 🧭 pfsense_dynamic_dns_rfc (Data Source)

Reads RFC DNS update configurations. Live appliance compatibility remains unverified.

## 🧪 Example Usage

```terraform
data "pfsense_dynamic_dns_rfc" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive API response. Use `jsondecode()` to access fields.

The response may include key material. Protect Terraform state and saved plans.
