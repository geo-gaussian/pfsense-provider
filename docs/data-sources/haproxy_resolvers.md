# 🧭 pfsense_haproxy_resolvers (Data Source)

Reads HAProxy DNS resolvers. Live appliance compatibility remains unverified.

## 🧪 Example Usage

```terraform
data "pfsense_haproxy_resolvers" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive API response. Use `jsondecode()` to access fields.

The response may identify network hosts. Protect Terraform state and saved plans.
