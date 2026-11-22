# 🧭 pfsense_interface_stats (Data Source)

Reads network interface traffic statistics and host addresses. The JSON result
is marked sensitive.

## 🧪 Example Usage

```terraform
data "pfsense_interface_stats" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain network details; protect them accordingly.
