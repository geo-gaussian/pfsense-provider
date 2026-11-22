# 🧭 pfsense_interface_statistics_widget (Data Source)

Reads packet, byte, error, and collision counts for configured interfaces.

## 🧪 Example Usage

```terraform
data "pfsense_interface_statistics_widget" "current" {}

output "interface_statistics" {
  value = jsondecode(data.pfsense_interface_statistics_widget.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields.
