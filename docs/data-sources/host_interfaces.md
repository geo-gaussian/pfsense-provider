# 🧭 pfsense_host_interfaces (Data Source)

Reads host network interfaces available to pfSense.

## 🧪 Example Usage

```terraform
data "pfsense_host_interfaces" "current" {}

output "host_interfaces" {
  value = jsondecode(data.pfsense_host_interfaces.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
