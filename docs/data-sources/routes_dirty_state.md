# 🧭 pfsense_routes_dirty_state (Data Source)

Reads whether routing configuration has pending changes. It does not apply
those changes.

## 🧪 Example Usage

```terraform
data "pfsense_routes_dirty_state" "current" {}

output "routes_dirty_state" {
  value = jsondecode(data.pfsense_routes_dirty_state.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields.
