# 🧭 pfsense_firewall_virtual_ip (Resource)

Manages a firewall virtual IP.

!> Experimental: mutations and reloads have only been tested against mock
servers. Appliance behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
resource "pfsense_firewall_virtual_ip" "service" {
  mode      = "ipalias"
  interface = "wan"
  subnet    = "198.51.100.20/32"
  descr     = "Service address"
}
```

## ⚙️ Argument Reference

- `id` (Optional, Computed) API virtual-IP identifier; changing it replaces
  the resource.
- `mode`, `interface`, `subnet`, and `descr` (Optional) Virtual-IP type,
  interface, address, and description.
- `vhid`, `advskew`, `advbase`, `carp_mode`, and `carp_peer` (Optional) CARP
  settings.
- `uniqid` (Optional) API uniqueness identifier.
- `password` and `password_confirm` (Optional, Sensitive) CARP password values.
  If the API omits them or returns empty values on read, the provider retains
  the existing state values. Protect Terraform state and saved plans.

Mutations initiate a firewall filter reload but do not confirm completion.

## 📥 Import

Import by API virtual-IP ID:

```shell
terraform import pfsense_firewall_virtual_ip.service 12345
```
