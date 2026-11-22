# 🧭 pfsense_dhcp_leases (Data Source)

Reads current DHCP leases, including client identifiers and network addresses.
The JSON result is marked sensitive.

## 🧪 Example Usage

```terraform
data "pfsense_dhcp_leases" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain client information; protect them accordingly.
