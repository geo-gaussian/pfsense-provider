# 🧭 pfsense_service_watchdog (Data Source)

Reads the services configured for watchdog monitoring.

## 🧪 Example Usage

```terraform
data "pfsense_service_watchdog" "current" {}

output "watched_services" {
  value = jsondecode(data.pfsense_service_watchdog.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
