# 🧭 pfsense_interfaces_status (Data Source)

Reads configured interface status and network details. The JSON result is
marked sensitive.

## 🧪 Example Usage

```terraform
data "pfsense_interfaces_status" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain interface details; protect them accordingly.
