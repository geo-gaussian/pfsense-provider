# 🧭 pfsense_services_status (Data Source)

Reads enabled and running status for system services.

## 🧪 Example Usage

```terraform
data "pfsense_services_status" "current" {}

output "services_status" {
  value = jsondecode(data.pfsense_services_status.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
