# 🧭 pfsense_haproxy_resolver_stats (Data Source)

Reads HAProxy DNS resolver statistics. Live appliance compatibility remains
unverified.

## 🧪 Example Usage

```terraform
data "pfsense_haproxy_resolver_stats" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive API response. Use `jsondecode()` to access fields.

The response may identify network hosts. Protect Terraform state and saved plans.
