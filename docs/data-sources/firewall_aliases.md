# pfsense_firewall_aliases (Data Source)

Reads metadata for user-defined firewall aliases on the controller's local
firewall. This data source does not write firewall configuration or request a
reload. Authentication may create a session or audit record.

## 🧪 Example Usage

```terraform
data "pfsense_firewall_aliases" "current" {}

output "firewall_aliases" {
  value = data.pfsense_firewall_aliases.current.aliases
}
```

## 📋 Attribute Reference

- `aliases` — List of user-defined alias metadata. If the API omits the list,
  this value is null and a warning is emitted. System aliases are not included.
  - `name` — Alias name.
  - `type` — Alias type, or null if omitted by the API.
  - `descr` — Alias description, or null if omitted by the API.
  - `truncated` — Whether the API truncated alias values, or null if omitted.

Alias contents are not returned. Terraform state and saved plans can contain
alias metadata; protect them accordingly.
