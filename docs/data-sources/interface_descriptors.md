# 🧭 pfsense_interface_descriptors (Data Source)

Reads interface descriptions grouped by interface type.

## 🧪 Example Usage

```terraform
data "pfsense_interface_descriptors" "current" {}

output "interface_descriptors" {
  value = jsondecode(data.pfsense_interface_descriptors.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
