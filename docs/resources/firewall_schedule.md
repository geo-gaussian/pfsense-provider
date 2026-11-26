# 🧭 pfsense_firewall_schedule (Resource)

Manages a firewall schedule.

!> Experimental: mutations and reloads have only been tested against mock
servers. Appliance behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
resource "pfsense_firewall_schedule" "weekdays" {
  name = "weekdays"
  descr = "Business hours"
  timerange = [{
    position   = "0"
    month      = "1-12"
    day        = "1-5"
    hour       = "08:00-17:00"
    rangedescr = "Weekday business hours"
  }]
}
```

## ⚙️ Argument Reference

- `id` (Optional, Computed) API schedule identifier; changing it replaces the
  resource.
- `name` (Required) Schedule name.
- `descr` (Optional) Schedule description.
- `schedlabel` (Optional) Schedule label.
- `timerange` (Optional) List of ranges with `position`, `month`, `day`, `hour`,
  and `rangedescr` fields.

The provider initiates a firewall filter reload after mutations but does not
wait for completion.

## 📥 Import

Import by API schedule ID:

```shell
terraform import pfsense_firewall_schedule.weekdays 12345
```
