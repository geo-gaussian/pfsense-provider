# 🧭 pfsense_gateway_groups (Data Source)

Reads configured gateway groups.

## 🧪 Example Usage

```terraform
data "pfsense_gateway_groups" "current" {}

output "gateway_groups" {
  value = jsondecode(data.pfsense_gateway_groups.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
