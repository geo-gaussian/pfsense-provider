# 🧭 pfsense_gateways_status (Data Source)

Reads gateway connectivity and monitor status, including network addresses.
The JSON result is marked sensitive.

## 🧪 Example Usage

```terraform
data "pfsense_gateways_status" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain network details; protect them accordingly.
