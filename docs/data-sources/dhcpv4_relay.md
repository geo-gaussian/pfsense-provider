# 🧭 pfsense_dhcpv4_relay (Data Source)

Reads DHCPv4 relay settings.

## 🧪 Example Usage

```terraform
data "pfsense_dhcpv4_relay" "current" {}

output "dhcpv4_relay" {
  value = jsondecode(data.pfsense_dhcpv4_relay.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
