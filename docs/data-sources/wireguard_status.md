# 🧭 pfsense_wireguard_status (Data Source)

Reads WireGuard operational status. Private keys, preshared keys, and
configuration objects are omitted from the JSON projection.

## 🧪 Example Usage

```terraform
data "pfsense_wireguard_status" "current" {}

output "wireguard_status" {
  value = jsondecode(data.pfsense_wireguard_status.current.json)
}
```

## 📋 Attribute Reference

- `json` — WireGuard status JSON without private keys, preshared keys, or
  configuration objects.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the status response; protect them accordingly.
