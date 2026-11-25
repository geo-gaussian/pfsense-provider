# 🧭 pfsense_firewall_pflow_exporter (Resource)

Manages a packet-flow exporter.

!> Experimental: mutations and reloads have only been tested against mock
servers. Appliance behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
resource "pfsense_firewall_pflow_exporter" "collector" {
  enable  = true
  src     = "192.0.2.1"
  srcport = 2055
  dst     = "198.51.100.20"
  dstport = 2055
  proto   = "udp"
  descr   = "Flow collector"
}
```

## ⚙️ Argument Reference

- `id` (Optional, Computed) API exporter identifier; changing it replaces the
  resource.
- `enable`, `src`, `srcport`, `dst`, `dstport`, `proto`, and `domain`
  (Optional) Exporter settings.
- `descr` (Optional) Exporter description.

Global packet-flow options are exposed by the
[`pfsense_firewall_pflow` data source](../data-sources/firewall_pflow.md).
Mutations initiate a firewall filter reload but do not confirm completion.

## 📥 Import

Import by API exporter ID:

```shell
terraform import pfsense_firewall_pflow_exporter.collector 12345
```
