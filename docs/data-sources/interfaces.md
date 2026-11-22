# 🧭 pfsense_interfaces (Data Source)

Reads basic information about configured network interfaces.

## 🧪 Example Usage

```terraform
data "pfsense_interfaces" "current" {}

output "interfaces" {
  value = jsondecode(data.pfsense_interfaces.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
