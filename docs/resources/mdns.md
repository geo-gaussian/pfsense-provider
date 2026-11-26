# 🔧 pfsense_mdns (Resource)

Manages the global multicast DNS service configuration.

!> This resource has only been tested against mock servers. Appliance
behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
resource "pfsense_mdns" "global" {
  enable             = true
  include_interfaces = ["lan"]
  reflection         = false
}
```

## ⚙️ Argument Reference

- `enable` (Optional) Enable the service.
- `carp_vip_intf` (Optional) CARP interface used for high availability.
- `include_interfaces` (Optional) Interfaces on which to listen.
- `exclude_interfaces` (Optional) Interfaces excluded from listening.
- `disable_ipv4`, `disable_ipv6` (Optional) Disable the relevant IP version.
- `reflection` (Optional) Reflect multicast DNS across networks.
- `publish` (Optional) Publishing options: `enable`, `addresses`, `hostinfo`,
  `workstation`, `domain`, `custom_hostname`, `custom_domain`,
  `aaaa_over_ipv4` and `a_over_ipv6`.
- `id` (Computed) Singleton identifier.

Destroy disables the service while retaining its other settings. Updates read
the current configuration first; concurrent writers may still overwrite one
another because requests are not locked across Terraform processes.

## 📥 Import

```shell
terraform import pfsense_mdns.global mdns
```
