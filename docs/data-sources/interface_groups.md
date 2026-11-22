# 🧭 pfsense_interface_groups (Data Source)

Reads configured interface groups.

## 🧪 Example Usage

```terraform
data "pfsense_interface_groups" "current" {}

output "interface_groups" {
  value = jsondecode(data.pfsense_interface_groups.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
