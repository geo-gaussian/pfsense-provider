# 🧭 pfsense_firewall_schedules (Data Source)

Reads configured firewall schedules.

## 🧪 Example Usage

```terraform
data "pfsense_firewall_schedules" "current" {}

output "firewall_schedules" {
  value = jsondecode(data.pfsense_firewall_schedules.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
