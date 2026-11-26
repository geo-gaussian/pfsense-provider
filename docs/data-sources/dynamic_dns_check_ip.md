# 🧭 pfsense_dynamic_dns_check_ip (Data Source)

Reads Dynamic DNS CheckIP services. Live appliance compatibility remains unverified.

## 🧪 Example Usage

```terraform
data "pfsense_dynamic_dns_check_ip" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive API response. Use `jsondecode()` to access fields.

The response may include credentials. Protect Terraform state and saved plans.
