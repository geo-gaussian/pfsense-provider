# pfsense_firewall_alias (Resource)

Manages a firewall alias on the controller's local firewall.

!> Experimental: create, update, delete, and firewall reloads have only been
tested against mock servers. Do not use this resource on a production firewall.

## 🧪 Example Usage

```terraform
resource "pfsense_firewall_alias" "example" {
  name    = "example"
  type    = "host"
  targets = [{ name = "host.example.test" }]
}
```

## ⚙️ Argument Reference

- `name` (Required) Alias name. Only letters, digits, and underscores are
  allowed. Changing it replaces the resource.
- `type` (Optional) One of `host`, `network`, `url`, `urltable`,
  `urltable_ports`, `port`, or `url_ports`.
- `address` (Optional) Space-separated addresses.
- `descr` (Optional) Alias description.
- `detail` (Optional) Alias detail.
- `tags` (Optional) Alias tags.
- `updatefreq` (Optional) Update frequency.
- `targets` (Optional) List of target objects:
  - `name` (Required) Target name.
  - `descr` (Optional) Target description.
  - `updatefreq` (Optional) Target update frequency.

Optional attributes can be populated by the API when not configured. Changes
initiate a firewall filter reload, but do not confirm it has completed. If a
write or reload fails, inspect the firewall configuration and initiate a
reload manually before retrying.

## 📥 Import

Import an existing alias by name:

```shell
terraform import pfsense_firewall_alias.example example
```
