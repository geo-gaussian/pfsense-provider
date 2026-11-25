# 🧭 pfsense_firewall_nat_npt (Resource)

Manages an IPv6 Network Prefix Translation (NPT) rule.

!> Experimental: mutations and reloads have only been tested against mock
servers. Appliance behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
resource "pfsense_firewall_nat_npt" "example" {
  interface = "wan"
  source = {
    address = "2001:db8:1::/48"
    type    = "network"
  }
  destination = {
    address = "2001:db8:2::/48"
    type    = "network"
  }
  descr = "NPT example"
}
```

## ⚙️ Argument Reference

- `id` (Optional, Computed) API rule identifier; changing it replaces the
  resource.
- `interface` (Optional) Firewall interface.
- `source` and `destination` (Optional) Address objects with `address`,
  `type`, and `not` fields.
- `descr` (Optional) Rule description.
- `disabled` (Optional) Whether the rule is disabled.

This API has no update operation, so changes replace the resource. The provider
initiates a firewall filter reload after mutations but does not wait for
completion.

## 📥 Import

Import by API rule ID:

```shell
terraform import pfsense_firewall_nat_npt.example 12345
```
