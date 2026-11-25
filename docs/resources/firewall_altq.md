# 🧭 pfsense_firewall_altq (Resource)

Manages an ALTQ root queue and its child queues.

!> Experimental: mutations and reloads have only been tested against mock
servers. Appliance behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
resource "pfsense_firewall_altq" "wan" {
  if_ident      = "wan"
  scheduler     = "HFSC"
  bandwidth     = "100"
  bandwidthtype = "Mb"
  queue = [{
    name        = "web"
    description = "Web traffic"
    bandwidth   = "80"
    bandwidthtype = "%"
  }]
}
```

## ⚙️ Argument Reference

- `if_ident`, `scheduler`, `bandwidth`, and `bandwidthtype` (Required) Root
  queue interface, scheduling algorithm, and bandwidth.
- `name` (Optional, Computed) API-generated queue identifier; use the computed
  value for import.
- `enabled`, `if_assigned_name`, `qlimit`, and `tbrsize` (Optional) Root queue
  settings.
- `queue` (Optional) Child queues with `name` and supported queue properties
  such as `priority`, `qlimit`, `description`, `bandwidth`, and HFSC limits.

Queue settings are managed together as one Terraform resource. Nested queues
are supported one level below the root. Mutations initiate a firewall filter
reload but do not confirm completion.

## 📥 Import

Import by API root-queue name:

```shell
terraform import pfsense_firewall_altq.wan <queue-name>
```
