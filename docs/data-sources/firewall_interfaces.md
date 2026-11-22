# 🧭 pfsense_firewall_interfaces (Data Source)

Reads the interfaces available for firewall rules.

## 🧪 Example Usage

```terraform
data "pfsense_firewall_interfaces" "current" {}

output "firewall_interfaces" {
  value = jsondecode(data.pfsense_firewall_interfaces.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
