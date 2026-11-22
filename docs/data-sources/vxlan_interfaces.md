# 🧭 pfsense_vxlan_interfaces (Data Source)

Reads configured VXLAN interfaces.

## 🧪 Example Usage

```terraform
data "pfsense_vxlan_interfaces" "current" {}

output "vxlan_interfaces" {
  value = jsondecode(data.pfsense_vxlan_interfaces.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
