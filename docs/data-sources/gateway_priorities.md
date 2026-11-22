# 🧭 pfsense_gateway_priorities (Data Source)

Reads configured gateways and their priority labels.

## 🧪 Example Usage

```terraform
data "pfsense_gateway_priorities" "current" {}

output "gateway_priorities" {
  value = jsondecode(data.pfsense_gateway_priorities.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
