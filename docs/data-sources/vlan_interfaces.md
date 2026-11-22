# 🧭 pfsense_vlan_interfaces (Data Source)

Reads configured VLAN interfaces.

## 🧪 Example Usage

```terraform
data "pfsense_vlan_interfaces" "current" {}

output "vlan_interfaces" {
  value = jsondecode(data.pfsense_vlan_interfaces.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
