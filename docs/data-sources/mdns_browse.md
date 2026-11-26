# 🧭 pfsense_mdns_browse (Data Source)

Browses multicast DNS services. Live appliance compatibility remains unverified.

## 🧪 Example Usage

```terraform
data "pfsense_mdns_browse" "current" {
  service = "_http._tcp"
}
```

## 📋 Attribute Reference

- `service` — Optional service type. Omit to browse without a service filter.
- `json` — Complete sensitive API response. Use `jsondecode()` to access fields.

The response may identify network hosts. Protect Terraform state and saved plans.
