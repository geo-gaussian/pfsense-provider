# 🧭 pfsense_system_status (Data Source)

Reads system and interface status details. The JSON result is marked sensitive
because it may include license and controller information.

## 🧪 Example Usage

```terraform
data "pfsense_system_status" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain system details; protect them accordingly.
