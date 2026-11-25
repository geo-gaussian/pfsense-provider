# 🧭 pfsense_firewall_nat_one_to_one (Resource)

Manages a one-to-one NAT rule.

!> Experimental: mutations and reloads have only been tested against mock
servers. Appliance behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
resource "pfsense_firewall_nat_one_to_one" "example" {
  interface = "wan"
  external = {
    address = "198.51.100.10"
  }
  destination = {
    address = "192.0.2.10"
  }
  descr = "One-to-one NAT example"
}
```

## ⚙️ Argument Reference

- `id` (Optional, Computed) API rule identifier; changing it replaces the
  resource.
- `interface` (Optional) Firewall interface.
- `external`, `destination`, and `source` (Optional) Address objects with
  `address`, `type`, `port`, and `not` fields.
- `descr` (Optional) Rule description.
- `disabled` and `nobinat` (Optional) Rule flags.
- `ipprotocol` (Optional) IP protocol.

This API has no update operation, so changes replace the resource. The provider
initiates a firewall filter reload after mutations but does not wait for
completion.

## 📥 Import

Import by API rule ID:

```shell
terraform import pfsense_firewall_nat_one_to_one.example 12345
```
