# 🧭 pfsense_openvpn_status (Data Source)

Reads active OpenVPN server and client status.

## 🧪 Example Usage

```terraform
data "pfsense_openvpn_status" "current" {}

output "openvpn_status" {
  value = jsondecode(data.pfsense_openvpn_status.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
