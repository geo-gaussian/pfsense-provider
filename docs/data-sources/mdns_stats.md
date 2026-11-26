# 🧭 pfsense_mdns_stats (Data Source)

Reads multicast DNS service state and records. Live appliance compatibility
remains unverified.

## 🧪 Example Usage

```terraform
data "pfsense_mdns_stats" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive API response. Use `jsondecode()` to access fields.

The response may identify network hosts. Protect Terraform state and saved plans.
