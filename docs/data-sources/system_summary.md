# 🧭 pfsense_system_summary (Data Source)

Reads the system name, operating system, architecture, and API version.

## 🧪 Example Usage

```terraform
data "pfsense_system_summary" "current" {}

output "system_summary" {
  value = jsondecode(data.pfsense_system_summary.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
