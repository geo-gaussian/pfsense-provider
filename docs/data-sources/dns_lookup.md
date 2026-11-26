# 🧭 pfsense_dns_lookup (Data Source)

Looks up DNS records for a host. Live appliance compatibility remains unverified.

## 🧪 Example Usage

```terraform
data "pfsense_dns_lookup" "example" {
  host = "example.test"
}
```

## 📋 Attribute Reference

- `host` — Required host name.
- `json` — Complete sensitive API response. Use `jsondecode()` to access fields.

The response may identify network hosts. Protect Terraform state and saved plans.
