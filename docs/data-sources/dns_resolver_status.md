# 🧭 pfsense_dns_resolver_status (Data Source)

Reads DNS Resolver query and server status statistics.

## 🧪 Example Usage

```terraform
data "pfsense_dns_resolver_status" "current" {}

output "dns_resolver_status" {
  value = jsondecode(data.pfsense_dns_resolver_status.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
