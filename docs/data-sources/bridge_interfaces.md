# 🧭 pfsense_bridge_interfaces (Data Source)

Reads configured bridge interfaces.

## 🧪 Example Usage

```terraform
data "pfsense_bridge_interfaces" "current" {}

output "bridge_interfaces" {
  value = jsondecode(data.pfsense_bridge_interfaces.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
