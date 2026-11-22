# 🧭 pfsense_interface_assignments (Data Source)

Reads configured interface assignments.

## 🧪 Example Usage

```terraform
data "pfsense_interface_assignments" "current" {}

output "interface_assignments" {
  value = jsondecode(data.pfsense_interface_assignments.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
