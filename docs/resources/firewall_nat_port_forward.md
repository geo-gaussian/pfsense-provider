# 🧭 pfsense_firewall_nat_port_forward (Resource)

Manages a port-forward NAT rule.

!> Experimental: mutations and reloads have only been tested against mock
servers. Appliance behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
resource "pfsense_firewall_nat_port_forward" "https" {
  interface = "wan"
  protocol  = "tcp"
  source = {
    address = "any"
  }
  destination = {
    address = "wanip"
    port    = "443"
  }
  target = {
    address = "192.0.2.10"
    port    = "443"
  }
  descr = "HTTPS service"
}
```

## ⚙️ Argument Reference

- `id` (Optional, Computed) API rule identifier; changing it replaces the
  resource.
- `interface`, `protocol`, and `ipprotocol` (Optional) Rule interface and
  protocol selectors.
- `source`, `destination`, and `target` (Optional) Address objects with
  `address`, `type`, `port`, and `not` fields.
- `descr` (Optional) Rule description.
- `disabled`, `nordr`, and `nosync` (Optional) Rule flags.
- `filter_rule_association` and `natreflection` (Optional) NAT behaviour
  settings.

The API response may include the associated filter-rule ID and audit timestamps;
these are populated from the API. Changes initiate a firewall filter reload but
do not confirm completion.

## 📥 Import

Import by API rule ID:

```shell
terraform import pfsense_firewall_nat_port_forward.example 12345
```
