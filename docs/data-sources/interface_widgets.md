# 🧭 pfsense_interface_widgets (Data Source)

Reads interface status and address details shown by the system dashboard. The
JSON result is marked sensitive.

## 🧪 Example Usage

```terraform
data "pfsense_interface_widgets" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain interface details; protect them accordingly.
