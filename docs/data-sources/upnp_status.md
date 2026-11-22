# 🧭 pfsense_upnp_status (Data Source)

Reads active UPnP mappings, including internal client addresses. The JSON
result is marked sensitive.

## 🧪 Example Usage

```terraform
data "pfsense_upnp_status" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain local network information; protect them accordingly.
