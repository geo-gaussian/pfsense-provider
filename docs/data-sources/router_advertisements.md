# 🧭 pfsense_router_advertisements (Data Source)

Reads IPv6 router advertisement settings.

## 🧪 Example Usage

```terraform
data "pfsense_router_advertisements" "current" {}

output "router_advertisements" {
  value = jsondecode(data.pfsense_router_advertisements.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
