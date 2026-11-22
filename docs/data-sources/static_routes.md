# 🧭 pfsense_static_routes (Data Source)

Reads configured static routes.

## 🧪 Example Usage

```terraform
data "pfsense_static_routes" "current" {}

output "static_routes" {
  value = jsondecode(data.pfsense_static_routes.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
