# 🧭 pfsense_dynamic_dns (Data Source)

Reads Dynamic DNS configurations. Live appliance compatibility remains unverified.

## 🧪 Example Usage

```terraform
data "pfsense_dynamic_dns" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive API response. Use `jsondecode()` to access fields.

The response may include credentials. Protect Terraform state and saved plans.
