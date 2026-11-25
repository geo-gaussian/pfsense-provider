# 🧾 pfsense_wireguard_tunnel_conf (Data Source)

Reads the rendered configuration for a WireGuard tunnel. The result can contain
private-key material.

!> The computed `json` value is sensitive, but Terraform state and saved plans
can still contain the rendered configuration. Protect them accordingly.

## 🧪 Example Usage

```terraform
data "pfsense_wireguard_tunnel_conf" "site" {
  name = "wg0"
}

output "wireguard_conf" {
  value     = jsondecode(data.pfsense_wireguard_tunnel_conf.site.json).conf
  sensitive = true
}
```

## ⚙️ Argument Reference

- `name` (Required) WireGuard tunnel name.

## 📋 Attribute Reference

- `json` (Computed, Sensitive) API response containing the rendered tunnel
  configuration.
