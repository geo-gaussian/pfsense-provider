# 🔧 pfsense_wireguard_settings (Resource)

Manages the global WireGuard settings singleton.

!> WireGuard operations have only been tested against mock servers. Appliance
behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
resource "pfsense_wireguard_settings" "global" {
  enable          = true
  keep_conf       = true
  interface_group = "all"
  hide_secrets    = true
  hide_peers      = false
}
```

## ⚙️ Argument Reference

- `enable` (Optional) Enable WireGuard.
- `keep_conf` (Optional) Keep the generated configuration.
- `resolve_interval` (Optional) DNS resolution interval.
- `resolve_interval_track` (Optional) Track DNS resolution intervals.
- `interface_group` (Optional) Interface group: `all`, `unassigned`, or `none`.
- `hide_secrets` (Optional) Hide secrets in generated configuration.
- `hide_peers` (Optional) Hide peer configuration.
- `id` (Computed) Singleton identifier.

The API has no operation to delete global settings. Destroying this Terraform
resource only relinquishes management; it does not reset or disable the
firewall's settings.

## 📥 Import

Import the singleton using its fixed ID:

```shell
terraform import pfsense_wireguard_settings.global wireguard
```
