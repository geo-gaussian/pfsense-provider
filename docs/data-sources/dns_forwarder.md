# 🧭 pfsense_dns_forwarder (Data Source)

Reads DNS Forwarder settings. The JSON result is marked sensitive because
custom options may contain arbitrary configuration.

## 🧪 Example Usage

```terraform
data "pfsense_dns_forwarder" "current" {}

output "dns_forwarder" {
  value     = jsondecode(data.pfsense_dns_forwarder.current.json)
  sensitive = true
}
```

## 📋 Attribute Reference

- `json` — Complete sensitive JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
