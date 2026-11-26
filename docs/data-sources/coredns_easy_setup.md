# 🧭 pfsense_coredns_easy_setup (Data Source)

Reads whether CoreDNS easy setup is required, without triggering setup.
Live appliance compatibility remains unverified.

## 🧪 Example Usage

```terraform
data "pfsense_coredns_easy_setup" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete API response. Use `jsondecode()` to access fields.
