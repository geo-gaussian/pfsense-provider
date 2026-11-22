# 🧭 pfsense_dhcpv6_relay (Data Source)

Reads DHCPv6 relay settings.

## 🧪 Example Usage

```terraform
data "pfsense_dhcpv6_relay" "current" {}

output "dhcpv6_relay" {
  value = jsondecode(data.pfsense_dhcpv6_relay.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
