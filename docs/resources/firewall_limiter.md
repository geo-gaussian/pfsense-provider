# 🧭 pfsense_firewall_limiter (Resource)

Manages a firewall limiter and its queues.

!> Experimental: mutations and reloads have only been tested against mock
servers. Appliance behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
resource "pfsense_firewall_limiter" "download" {
  name        = "download"
  enabled     = true
  description = "Download bandwidth"
  bandwidth = [{
    bw      = 100
    bwscale = "Mb"
  }]
  mask = "dstaddress"
  aqm  = "codel"
  sched = "fq_codel"
  queue = [{
    name        = "interactive"
    enabled     = true
    description = "Interactive traffic"
  }]
}
```

## ⚙️ Argument Reference

- `name` (Required) Limiter name; changing it replaces the resource.
- `enabled`, `description`, `mask`, `maskbits`, and `maskbitsv6` (Optional)
  Limiter state and address masking.
- `bandwidth` (Optional) Bandwidth entries with `bw`, `bwscale`, and `bwsched`.
- `aqm`, `sched`, `qlimit`, `delay`, `plr`, and `buckets` (Optional) Queue
  management and scheduling settings.
- `queue` (Optional) Nested limiter queues. Queue attributes follow the
  corresponding API model, including `name`, `enabled`, `mask`, `aqm`, `sched`,
  and queue limits.
- `param_*` and `pie_*` (Optional) Algorithm-specific parameters.

Queues are managed together with their parent limiter. Mutations initiate a
firewall filter reload but do not confirm completion.

## 📥 Import

Import by limiter name:

```shell
terraform import pfsense_firewall_limiter.download download
```
