# 🧰 pfsense_wireguard_config (Data Source)

Reads the full WireGuard configuration, including private-key and preshared-key
fields when returned by the API.

!> The computed `json` value is sensitive, but Terraform state and saved plans
can still contain the underlying configuration. Protect them accordingly.

## 🧪 Example Usage

```terraform
data "pfsense_wireguard_config" "current" {}

output "wireguard_tunnels" {
  value     = jsondecode(data.pfsense_wireguard_config.current.json).tunnels
  sensitive = true
}
```

## 📋 Attribute Reference

- `json` (Computed, Sensitive) Complete response from the WireGuard
  configuration endpoint. Use `jsondecode()` to select fields.
