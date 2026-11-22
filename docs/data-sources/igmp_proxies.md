# 🧭 pfsense_igmp_proxies (Data Source)

Reads configured IGMP proxy settings.

## 🧪 Example Usage

```terraform
data "pfsense_igmp_proxies" "current" {}

output "igmp_proxies" {
  value = jsondecode(data.pfsense_igmp_proxies.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
