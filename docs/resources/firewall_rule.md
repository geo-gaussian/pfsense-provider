# 🧭 pfsense_firewall_rule (Resource)

Manages a firewall filter rule on an interface.

!> Experimental: mutations and reloads have only been tested against mock
servers. Appliance behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
resource "pfsense_firewall_rule" "https" {
  interface  = "lan"
  type       = "pass"
  ipprotocol = "inet"
  protocol   = "tcp"
  descr      = "Allow HTTPS to service"
  source = {
    any = true
  }
  destination = {
    address = "192.0.2.10"
    port    = "443"
  }
}
```

## ⚙️ Argument Reference

- `id` (Optional, Computed) API rule identifier.
- `interface` (Optional) Interface containing the rule. Changing the interface
  replaces the resource.
- `type`, `ipprotocol`, `protocol`, `direction`, `gateway`, and `sched`
  (Optional) Rule action and matching settings.
- `source` and `destination` (Optional) Address/port objects; use `any`,
  `address`, `network`, `port`, `not`, or `alias_id` as appropriate.
- `descr` (Optional) Rule description.
- Other optional fields support the API's logging, state, TCP flags, tagging,
  limits, queue, limiter, and NAT options.

The API rule model is exposed as typed Terraform attributes. API-generated
labels, state counters, and audit timestamps are not configurable. Imports use
`interface/id`, for example `lan/12345`. Mutations initiate a firewall filter
reload but do not confirm completion.

## 📥 Import

```shell
terraform import pfsense_firewall_rule.https lan/12345
```
