# 🧭 pfsense_dirty_subsystems (Data Source)

Reads which subsystem configurations have pending changes.

## 🧪 Example Usage

```terraform
data "pfsense_dirty_subsystems" "current" {}

output "pending_subsystems" {
  value = jsondecode(data.pfsense_dirty_subsystems.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

This data source only reports pending changes; it does not apply them. Use
`jsondecode()` to access response fields.
