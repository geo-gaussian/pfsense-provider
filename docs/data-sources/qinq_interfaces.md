# 🧭 pfsense_qinq_interfaces (Data Source)

Reads configured QinQ interfaces.

## 🧪 Example Usage

```terraform
data "pfsense_qinq_interfaces" "current" {}

output "qinq_interfaces" {
  value = jsondecode(data.pfsense_qinq_interfaces.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
