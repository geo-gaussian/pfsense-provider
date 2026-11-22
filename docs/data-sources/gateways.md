# 🧭 pfsense_gateways (Data Source)

Reads configured system gateways.

## 🧪 Example Usage

```terraform
data "pfsense_gateways" "current" {}

output "gateways" {
  value = jsondecode(data.pfsense_gateways.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
